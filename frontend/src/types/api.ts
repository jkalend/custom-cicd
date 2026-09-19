export type PipelineStatus = 'pending' | 'running' | 'success' | 'failed' | 'cancelled' | 'never_run';

export interface PipelineStep {
  name: string;
  description: string;
  command: string;
  timeout: number;
  retry_count?: number;
  continue_on_error?: boolean;
  status?: PipelineStatus;
  output?: string;
  error?: string;
  start_time?: string;
  end_time?: string;
  ai_analysis?: AIAnalysis;
}

/** Jev analysis attached to a failed step by the backend. */
export interface AIAnalysis {
  classification: {
    kind: 'flaky' | 'build' | 'config' | 'infra' | 'code';
    retri: boolean;
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

export interface ApiResponse<T = any> {
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

// --- AI layer (Jev) ------------------------------------------------------------

export interface AIStatus {
  configured: boolean;
  gateway: string;
  model: string;
}

export interface Decision {
  id: string;
  ts: number;
  provider: 'jev' | 'heuristic';
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
