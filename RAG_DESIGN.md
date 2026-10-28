# Diseño del RAG de reglas (borrador)

Resumen de lo discutido el 2026-10-28. Nada de esto está implementado; las decisiones marcadas
como **abiertas** siguen pendientes, y las tomadas se revisan al implementar cada parte.

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
- **Todo es automático:** la única fuente es el sync del Rules Hub. **No hay carga manual**;
  lo que hoy está en `corpus/` y no está en el hub queda fuera del corpus.

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

**El pipeline es determinístico** (cron, sin LLM). Un agente LLM interviene solo cuando una
corrida termina con errores o chequeos de calidad que no pasan: ver §9.

**Lectura del hub:** playriftbound.com es Next.js, así que se lee el JSON de `__NEXT_DATA__`
con `gjson`, igual que la galería de cartas en `ingest`. Spike pendiente: confirmar que el
texto de patch notes y errata viene ahí.

## 5. Costo

Medido el 2026-10-28 (caracteres/4 ≈ tokens):

| Documento | Caracteres | Tokens aprox. |
|---|---|---|
| Core Rules | 375k | ~94k |
| Tournament Rules | 166k | ~42k |
| Patch notes + erratas (estimado) | ~150k | ~37k |
| ~~`corpus/` manual~~ (descartado, sin carga manual) | ~~252k~~ | ~~~63k~~ |
| **Total con solapamiento (+25%)** | | **~220k** |

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

- tablas y migraciones propias en el schema `rules` (§8);
- el cron diario del Rules Hub;
- el agente de revisión (§9);
- `POST /search` (§7).

**Esta API:**

- un `RulesSearchService` que llama a Go por HTTP, con timeout corto;
- una tool del agente de chat que lo usa;
- `Settings`: URL del servicio y secreto interno (llamada saliente, como dolarapi; la regla
  de "no API key" aplica a los clientes de esta API).

**Servicio propio** (`riftplay_embedding_service`), no dentro de `ingest`: son dos
responsabilidades distintas.

## 7. Contrato de `POST /search`

**Red:** privada en Railway (`*.railway.internal`, sin dominio público) y además
`Authorization: Bearer $INTERNAL_API_SECRET` (mín. 32 caracteres, comparación en tiempo
constante, nunca se loguea). `GET /healthz` queda sin auth.

```
POST /search
{
  "query": "¿puedo jugar dos runas por turno?",
  "k": 4,                                   // opcional, default 4, máx. 10
  "documents": ["core-rules", "errata/*"]   // opcional; identidad lógica o prefijo/*
}

200 OK
{
  "model": "text-embedding-3-small",
  "results": [
    {
      "id": "0193f...",
      "document": "core-rules",
      "title": "Core Rules",
      "source_url": "https://cmsassets.rgpub.io/...pdf",
      "rule": "357.2",          // null fuera del Core Rules
      "text": "...",
      "score": 0.82             // similitud coseno
    }
  ]
}
```

| Caso | Respuesta |
|---|---|
| `query` vacía, `k` fuera de rango, identidad inválida | `400 {"error": "..."}` |
| Falta el secreto o es incorrecto | `401 {"error": "unauthorized"}` |
| OpenAI falló o timeout | `502 {"error": "embedding provider"}` |
| Postgres no responde | `503 {"error": "database"}` |
| Nada encontrado / corpus vacío | `200` con `results: []` |

Python trata 4xx como bug propio y 5xx como fail open.

## 8. Base de datos

Mismo Postgres, **schema propio `rules`** con extensión `vector`. Alembic ignora el schema
entero (filtro por schema, no por lista de tablas). Python nunca lo lee.

```
rules.documents
  id            text PK          -- 'core-rules', 'errata/origins'
  kind          text NOT NULL    -- core | tournament | patch | errata | legality
  title         text NOT NULL
  source_url    text
  etag          text
  last_modified text
  content_hash  text NOT NULL    -- sha256 del texto extraído
  checked_at    timestamptz NOT NULL
  updated_at    timestamptz NOT NULL

rules.fragments
  id              uuid PK (v7)
  document_id     text FK → documents ON DELETE CASCADE
  ordinal         int NOT NULL
  rule            text           -- '357.2' o null
  text            text NOT NULL
  embedding       vector(1536) NOT NULL
  embedding_model text NOT NULL
  UNIQUE (document_id, ordinal)
  INDEX hnsw (embedding vector_cosine_ops)

rules.sync_runs
  id, started_at, finished_at, status (ok | failed), checked, changed text[], error
```

Una versión nueva con hallazgos queda **pendiente** y `/search` sigue sirviendo la aprobada
(§9). Cómo se modela "pendiente" (columna de estado o tabla aparte) se define al implementar.

## 9. Agente de revisión

El pipeline normal no usa LLM. El agente corre **solo** cuando una corrida termina con
hallazgos; los días normales no gasta tokens.

**Disparadores:**

- **error:** descarga, parseo u OpenAI fallaron tras los reintentos;
- **salvaguarda:** 0 documentos, falta `core-rules`, sección desconocida en el hub;
- **calidad:** las reglas detectadas cayeron >10% contra la versión anterior, fragmentos
  vacíos o gigantes, números de regla fuera de orden, golden queries que fallan.

**Golden queries:** archivo versionado en el repo (~20 preguntas en español, como preguntan
los usuarios) con el documento y la regla esperados en el top-k. Umbral: ≥ 90% de aciertos y
sin caída contra la versión anterior. Sirven también como test de integración y para comparar
modelos.

**Acciones (allowlist, nunca borra):**

| Solo | Pregunta al usuario |
|---|---|
| `retry_document(id)` | Aprobar con chequeos fallando |
| `rechunk(id, strategy)` (reglas ↔ 1000/200) | Borrar un documento que desapareció del hub |
| `run_golden_queries(id)` | Agregar un dominio permitido |
| `approve_pending(id)`, solo si **todos** los chequeos pasan | Cualquier cosa fuera de la allowlist |
| `ignore_section(name)` | |

Nunca: SQL libre, fetch a dominios no permitidos, borrar.

**Qué ve:** el reporte estructurado de la corrida (errores, conteos, diffs de métricas, golden
queries fallidas) y, si lo pide con `read_sample(document_id, ordinal, n≤3)`, muestras de
hasta 2000 caracteres envueltas como texto no confiable. Nunca el documento entero.

**Cómo pregunta:** un bot de chat (Telegram o Discord) con botones, uno por opción cerrada,
detrás de una interfaz `Notifier`. El estado vive en `rules.agent_reviews` (diagnóstico,
opciones, estado, respuesta, quién y cuándo, tokens usados).

**Sin respuesta:** se sigue sirviendo la versión aprobada, recordatorio diario, nunca expira
ni se aplica nada por timeout. Si el hallazgo persiste en el sync siguiente se actualiza la
misma review; si se resolvió solo, se cierra. Un documento nuevo sin versión anterior no se
sirve hasta que haya respuesta.

**Implementación:** en Go (`internal/review/`), loop de tools propio sobre la Messages API de
Anthropic con `net/http`. Config `ANTHROPIC_API_KEY` y `REVIEW_MODEL`; máx. 10 turnos y 5 min
por review.

## 10. Stack y orden de implementación

| Área | Elección |
|---|---|
| Postgres | `jackc/pgx/v5` + `pgxpool`, queries con `sqlc`, `pgvector/pgvector-go` |
| Migraciones | `.sql` embebidos + `cmd/migrate` propio (copiado de `ingest`) |
| Cron | River (cola `sync`, `MaxWorkers` 1, diario UTC + al arrancar), schema `river_embedding`; `robfig/cron/v3` |
| Hub | `tidwall/gjson` sobre `__NEXT_DATA__` |
| PDF | `pdftotext -layout` (poppler-utils en la imagen) por exec, con timeout y límite de descarga |
| Embeddings | Interfaz `Embedder`; cliente OpenAI propio con `net/http`, lotes de 100, reintentos en 429/5xx. Config `OPENAI_API_KEY`, `EMBEDDING_MODEL` |
| Tests | Integración contra `pgvector/pgvector` en Docker (local y CI), skip sin `TEST_DATABASE_URL`; unit tests con fixtures en `testdata/` |

**Orden:**

0. Spikes (sin código de producción): `pdftotext` sobre el Core Rules real y
   `__NEXT_DATA__` del hub.
1. Contrato de `/search` publicado para que Python avance en paralelo.
2. Config + migración `0001` (schema `rules`) + `cmd/migrate`.
3. `Embedder` + fragmentación por regla y por tamaño.
4. Sync del hub (River, detección de cambios, `sync_runs`).
5. `POST /search` + auth.
6. Agente de revisión + golden queries + `Notifier`.
7. Dockerfile con poppler-utils.

## 11. Decisiones abiertas

1. **Si Go no responde:** propuesta de fail open. La tool devuelve "búsqueda no disponible" y
   el agente responde aclarándolo, sin 503 sobre todo el chat.
2. **Spike de PDF:** confirmar que `pdftotext` saca los números de regla limpios. La
   fragmentación por regla depende de eso.
3. **Spike del hub:** confirmar que `__NEXT_DATA__` trae el texto de patch notes y errata.
4. **Canal del bot:** Telegram o Discord.
5. **Stack en `CLAUDE.md`:** sumar las dependencias del §10 a la tabla cuando se agreguen.
6. **Limpieza en la API Python:** `EMBEDDING_DIM = 384` y la tabla de la `0001` quedan solo por
   la migración vieja; la tabla nueva no los reutiliza.
