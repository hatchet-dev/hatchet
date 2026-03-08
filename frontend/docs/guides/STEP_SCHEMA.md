# Guide Step Schema

Shared step names used across Python, TypeScript, Go, and Ruby examples.
Snippet keys follow: `snippets.<lang>.guides.<guide_slug>.<file>.<step>`.

## human-in-the-loop
- `step_01_define_approval_task` - Durable task that proposes action
- `step_02_wait_for_event` - WaitForEvent for approval key
- `step_03_push_approval_event` - Push event from frontend/API
- `step_04_run_worker` - Worker registration and start

## ai-agents
- `step_01_define_agent_task` - Durable task with reasoning loop
- `step_02_reasoning_loop` - LLM call, tool execution, loop
- `step_03_stream_response` - put_stream for token streaming
- `step_04_run_worker` - Worker with concurrency control

## batch-processing
- `step_01_define_parent_task` - Parent workflow with batch input
- `step_02_fan_out_children` - Spawn child per item
- `step_03_process_item` - Child task processes single item
- `step_04_run_worker` - Worker with parent and child workflows

## document-processing
- `step_01_define_dag` - DAG workflow: ingest -> parse -> extract -> validate
- `step_02_parse_stage` - Parse stage (mock OCR)
- `step_03_extract_stage` - Extract stage (mock LLM)
- `step_04_run_worker` - Worker with DAG workflows

## event-driven
- `step_01_define_event_task` - Task triggered by event
- `step_02_register_event_trigger` - onEvents / event trigger
- `step_03_push_event` - Push event to trigger task
- `step_04_run_worker` - Worker registration

## llm-pipelines
- `step_01_define_pipeline` - DAG with prompt -> generate -> validate
- `step_02_prompt_task` - Build prompt (mock LLM)
- `step_03_validate_task` - Validate and retry on failure
- `step_04_run_worker` - Worker with rate limit

## rag-and-indexing
- `step_01_define_ingest_task` - Ingest documents
- `step_02_chunk_task` - Fan out, chunk per document
- `step_03_embed_task` - Embed chunks (mock)
- `step_04_run_worker` - Worker with rate limit

## scheduled-jobs
- `step_01_define_cron_task` - Cron-triggered task
- `step_02_schedule_one_time` - One-time scheduled run
- `step_03_run_worker` - Worker with cron workflow

## streaming
- `step_01_define_streaming_task` - Task that emits chunks
- `step_02_emit_chunks` - put_stream in worker
- `step_03_subscribe_client` - subscribe_to_stream on client
- `step_04_run_worker` - Worker registration

## webhook-processing
- `step_01_define_webhook_task` - Task triggered by webhook
- `step_02_register_webhook` - Webhook trigger config
- `step_03_process_payload` - Process webhook payload
- `step_04_run_worker` - Worker registration

## evaluator-optimizer
- `step_01_define_tasks` - Generator and evaluator child tasks
- `step_02_optimization_loop` - Durable task that loops generate → evaluate → feedback
- `step_03_run_worker` - Worker registration

## routing
- `step_01_classify_task` - Classification task (LLM or rule-based)
- `step_02_specialist_tasks` - Specialist handler tasks (support, sales, default)
- `step_03_router_task` - Durable router task with if/else + RunChild
- `step_04_run_worker` - Worker registration

## multi-agent
- `step_01_specialist_agents` - Specialist workflows (research, writing, code)
- `step_02_orchestrator_loop` - Durable orchestrator reasoning loop
- `step_03_run_worker` - Worker registration

## web-scraping
- `step_01_scrape_task` - Scrape a single URL (with retries)
- `step_02_fan_out_scrape` - Fan out to scrape multiple URLs
- `step_03_cron_refresh` - Cron workflow to refresh scrapes on schedule
- `step_04_run_worker` - Worker registration

## web-scraping
- `step_01_define_scrape_task` - Task that scrapes a URL (Firecrawl, Playwright, etc.)
- `step_02_process_content` - Extract/transform scraped content (optionally with LLM)
- `step_03_cron_workflow` - Cron workflow to refresh scrapes on a schedule
- `step_04_run_worker` - Worker registration

## parallelization
- `step_01_parallel_tasks` - Tasks that run concurrently (content, safety, evaluate)
- `step_02_sectioning` - Sectioning pattern: different concerns in parallel
- `step_03_voting` - Voting pattern: same evaluation N times, aggregate
- `step_04_run_worker` - Worker registration
