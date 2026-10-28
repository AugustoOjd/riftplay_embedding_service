# Diseño del RAG de reglas (borrador)

Resumen de lo discutido el 2026-10-28. Nada de esto está implementado; las decisiones marcadas
como **abiertas** siguen pendientes.

## 1. Búsqueda semántica vs. RAG

- **Búsqueda semántica:** embedding de la consulta → similitud en pgvector → top-k fragmentos.
  Sin LLM.
- **RAG:** la búsqueda semántica + un LLM que redacta la respuesta con esos fragmentos y cita.
- La búsqueda es la base y se prueba sola; el RAG es una tool del agente de chat
  (`app/llm/`) que la usa.
- Del tutorial de Deep Agents (un subagente por fragmento) **no** se adopta el análisis por
  fragmento: multiplica tokens contra el presupuesto de `token_usages`. Con k≈4 fragmentos
  de ~1000 caracteres alcanza con pasarlos directo al contexto.

## 2. Modelo de embeddings

**Elegido:** OpenAI `text-embedding-3-small`, por API. **Alternativa:** Voyage, si la
calidad no alcanza.

- El corpus está en **inglés** y los usuarios preguntan en **español**, así que el modelo
  tiene que ser multilingüe.
- **Se descarta volver al modelo local** (`multilingual-e5-small` + torch): causó el OOM en
  Railway (reportes 2026-09-05 y 2026-09-06).
- Cada fragmento guarda su `embedding_model`. Mezclar vectores de dos modelos rompe la
  búsqueda sin dar ningún error.
- Si fallan los nombres de cartas o la jerga del juego, más adelante se puede sumar búsqueda
  híbrida con el full-text de Postgres.

## 3. Carga de documentos

- **Proceso síncrono, sin guardar el archivo original:** extraer → fragmentar → embeber →
  reemplazar en **una transacción**. El texto ya queda en los fragmentos.
- **Idempotente por identidad lógica + `content_hash`:**
  - mismo hash → no hace nada;
  - hash distinto → reemplaza los fragmentos de ese documento;
  - identidad nueva → lo crea.
- Solo cargan administradores. Además del sync automático, queda una carga manual para
  documentos que no están en el hub, como las guías actuales de `corpus/`.

## 4. Sincronización diaria con el Rules Hub

Fuente: <https://playriftbound.com/en-us/rules-hub/>

| Sección | Formato | Comportamiento |
|---|---|---|
| Core Rules | PDF (142 págs.) | Se **reemplaza**; la URL es un hash y cambia con cada versión |
| Tournament Rules | PDF (53 págs., 17,5 MB) | Se reemplaza |
| Patch Notes | HTML, uno por set | Se **agregan** (o se reemplazan si se editan) |
| Errata | HTML, uno por set | Igual que Patch Notes |
| Legalidad (Constructed y 2v2) | HTML en la misma página | Se reemplaza |
| Related Articles, Judges | YouTube, FAQ, UVS | **No se indexan** |

**Identidad lógica, no URL:** `core-rules`, `tournament-rules`, `patch-notes/<set>`,
`errata/<set>`, `legality/<formato>`. Con "saltear lo existente y agregar lo nuevo", un
Core Rules nuevo convive con el viejo y el bot cita reglas derogadas.

**Detección de cambios, de lo más barato a lo más caro:**

1. Bajar el hub y listar los documentos de las secciones permitidas.
2. ¿URL distinta a la guardada? → hay versión nueva.
3. ¿Misma URL? → pedido condicional con `ETag`/`Last-Modified`; un 304 significa que no
   cambió.
4. Si hubo que bajarlo, comparar el hash del **texto extraído**, no de los bytes.
5. Solo si el hash cambió: fragmentar, embeber y reemplazar.

**Salvaguardas:**

- **Nunca borrar porque el scrape vino vacío.** Si aparecen 0 documentos, o falta uno fijo
  como `core-rules`, el job falla y lo loguea.
- **Dominios permitidos:** `playriftbound.com`, `riftbound.leagueoflegends.com` y
  `cmsassets.rgpub.io`.
- **Log de cada corrida:** qué se revisó, qué cambió y qué falló.
- **Inyección de prompt:** fuente oficial con dominios restringidos, así que el riesgo es
  aceptable sin revisión humana, pero cada cambio queda registrado.

**Fragmentación:** Core Rules **por número de regla**, con el número en la metadata para que
el bot cite "regla 357.2". El resto, por tamaño (1000/200).

**No hace falta un agente LLM:** es un job determinístico (cron).

## 5. Costo

Medido el 2026-10-28 (caracteres/4 ≈ tokens):

| Documento | Caracteres | Tokens aprox. |
|---|---|---|
| Core Rules | 375k | ~94k |
| Tournament Rules | 166k | ~42k |
| Patch notes + erratas (estimado) | ~150k | ~37k |
| `corpus/` manual | 252k | ~63k |
| **Total con solapamiento (+25%)** | | **~300k** |

Con `text-embedding-3-small` (USD 0,02 por millón de tokens; **verificar precio**), una
corrida completa cuesta ~USD 0,006: reindexar todo cada semana sale unos **USD 0,30 al año**.
El costo no es plata sino:

- **IDs de fragmentos:** reindexar todo los recrea y rompe lo que los referencie.
- **Memoria:** parsear un PDF de 17,5 MB. Por eso nunca dentro de un request de la API.

Se mantiene el chequeo diario con skip por hash. El reindexado completo es manual y solo se
hace al cambiar de modelo o de forma de fragmentar.

## 6. Arquitectura: servicio de embeddings en Go

Mismo patrón que `ingest`: Go es dueño del dato y esta API lo consume.

**Opción elegida (B): Go expone `/search`** y es el único que conoce el modelo.

| | A. Base compartida | **B. `/search` en Go** |
|---|---|---|
| Quién embebe la consulta | Python | Go |
| Dueño del modelo | Los dos (acoplados) | Solo Go |
| Cambiar de modelo | Deploy coordinado | Solo deploy de Go |

**Servicio Go:**

- tablas y migraciones propias: documentos (identidad, URL, hash, ETag, `checked_at`) y
  fragmentos (vector, `embedding_model`, número de regla);
- el cron diario del Rules Hub;
- la carga manual para admins;
- `POST /search {query, k}` → fragmentos con `source`, número de regla y puntaje.

**Esta API:**

- un `RulesSearchService` que llama a Go por HTTP, con timeout corto;
- una tool del agente de chat que lo usa;
- `Settings`: URL del servicio y secreto interno (llamada saliente, como dolarapi; la regla
  de "no API key" aplica a los clientes de esta API).

## 7. Decisiones abiertas

1. **Contrato de `/search`:** qué recibe y qué devuelve cada fragmento. Es el primer paso:
   permite construir los dos lados en paralelo.
2. **Si Go no responde:** propuesta de fail open. La tool devuelve "búsqueda no disponible" y
   el agente responde aclarándolo, sin 503 sobre todo el chat.
3. **Red:** privada en Railway, sin URL pública, con el secreto como defensa adicional.
4. **Base de datos:** misma base (Alembic debe ignorar las tablas, como con
   `ingest_owned_tables()`) o base aparte. Con B, Python no las lee nunca.
5. **Parsing de PDF en Go:** probar `pdftotext` (poppler) con el Core Rules real antes de
   comprometerse. La fragmentación por regla depende de que el texto salga limpio.
6. **Servicio nuevo o dentro de `ingest`:** ahorra un deploy, pero mezcla dos
   responsabilidades.
7. **Limpieza en este repo:** `EMBEDDING_DIM = 384` y la tabla de la `0001` quedan solo por la
   migración vieja; la tabla nueva no los reutiliza.
