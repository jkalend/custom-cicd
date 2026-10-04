export type PipelineStatus = 'pending' | 'running' | 'success' | 'failed' | 'cancelled' | 'skipped' | 'never_run';

export interface PipelineStep {
  name: string;
  description: string;
  command: string;
  timeout: number;
  retry_count?: number;
  continue_on_error?: boolean;
  depends_on?: string[];
  artifacts?: string[];
  status?: PipelineStatus;
  output?: string;
  error?: string;
  start_time?: string;
  end_time?: string;
  ai_analysis?: AIAnalysis;
}

/** Laya analysis attached to a failed step by the backend. */
export interface AIAnalysis {
  classification: {
    kind: 'flaky' | 'build' | 'config' | 'infra' | 'code';
    retri: boolean;
    retry?: boolean;
    retry_probability?: number | null;
    severity?: number | null;
    summary: string;
  };
  log_triage: {
    severity?: number | null;
    action: 'ignore' | 'investigate' | 'restart' | 'page';
    page_probability: number;
  };
}

export interface Pipeline {
  id: string;
  name: string;
  version: string;
  description?: string;
  status: PipelineStatus;
  created_at: string;
  active_runs?: number;
  total_runs?: number;
  last_run_at?: string;
  variables?: Record<string, string>;
  steps: PipelineStep[];
}

export interface PipelineRun {
  id: string;
  pipeline_id: string;
  name: string;
  status: PipelineStatus;
  created_at: string;
  started_at?: string;
  finished_at?: string;
  total_duration?: number;
  steps?: PipelineStep[];
}

export interface CreatePipelineRequest {
  name: string;
  version: string;
  description?: string;
  variables?: Record<string, string>;
  steps: PipelineStep[];
}

export interface ApiResponse<T = unknown> {
  data?: T;
  error?: string;
}

export interface CreatePipelineResponse {
  pipeline_id: string;
  status: string;
}

export interface CreateAndRunPipelineResponse {
  pipeline_id: string;
  run_id: string;
  status: string;
}

export interface RunPipelineResponse {
  run_id: string;
  status: string;
}

// --- AI layer (Laya) ------------------------------------------------------------

export interface AIStatus {
  configured: boolean;
  gateway: string;
  model: string;
}

export interface Decision {
  id: string;
  ts: number;
  provider: 'laya' | 'heuristic';
  latency_ms: number;
  context: { module: string; [key: string]: unknown };
  state: Record<string, unknown>;
  questions: Record<string, { type: string; instructions: string; criteria?: unknown }>;
  answers: Record<string, Record<string, unknown>>;
  error?: string;
}

export interface Notification {
  id: string;
  ts: string;
  run_id: string;
  pipeline_name: string;
  run_status: PipelineStatus;
  event_type: string;
  title: string;
  message: string;
  level: 'ignore' | 'defer' | 'notify' | 'urgent';
  urgent_probability: number;
}

export interface TriageResult {
  severity?: number | null;
  action: 'ignore' | 'investigate' | 'restart' | 'page';
  page_probability: number;
}

export interface IssueRouteResult {
  is_bug: number;
  component: string;
  severity: number;
  labels: string[];
}

export interface ArtifactInfo {
  name: string;
  step_name: string;
  size: number;
  path: string;
}

export interface FixSuggestion {
  step_name: string;
  cause: string;
  hypothesis: string;
  suggested_command: string;
  can_auto_apply: boolean;
  confidence: number;
  provider: string;
}

export interface FeedbackPayload {
  decision_id?: string;
  rating: 'positive' | 'negative';
  comment?: string;
  user?: string;
}

export interface FeedbackEntry extends FeedbackPayload {
  id: string;
  ts: number;
}

export interface LogEvent {
  run_id: string;
  step_index: number;
  step_name: string;
  stream: 'stdout' | 'stderr' | 'system';
  line: string;
  timestamp: string;
}

