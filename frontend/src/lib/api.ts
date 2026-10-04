// API client for the CI/CD backend. This file was imported by every page
// but never committed to the original repo (frontend couldn't build from a
// fresh clone) — restored and extended with the AI-layer endpoints.

import type {
  AIStatus,
  ArtifactInfo,
  CreateAndRunPipelineResponse,
  CreatePipelineRequest,
  CreatePipelineResponse,
  Decision,
  FeedbackEntry,
  FeedbackPayload,
  FixSuggestion,
  IssueRouteResult,
  LogEvent,
  Notification,
  Pipeline,
  PipelineRun,
  PipelineStep,
  RunPipelineResponse,
  TriageResult,
} from '@/types/api';

export class ApiError extends Error {
  constructor(
    message: string,
    public status: number
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(path, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });

  let body: unknown;
  try {
    body = await response.json();
  } catch {
    body = null;
  }

  const errObj = body as { error?: string; message?: string } | null;
  const wrapped = body as { success?: boolean; data?: T; error?: string } | null;
  if (wrapped && typeof wrapped === 'object' && 'success' in wrapped) {
    if (!wrapped.success) {
      throw new ApiError(wrapped.error ?? `Request failed (${response.status})`, response.status);
    }
    return wrapped.data as T;
  }

  if (!response.ok) {
    const errorMsg = errObj?.error ?? errObj?.message ?? `Request failed (${response.status})`;
    throw new ApiError(errorMsg, response.status);
  }

  return body as T;
}

export const apiClient = {
  // Health
  health: () => request<{ status: string; timestamp: string; agent_status: string }>('/api/health'),

  // Pipelines
  listPipelines: () => request<Pipeline[]>('/api/pipelines'),
  getPipeline: (id: string) => request<Pipeline>(`/api/pipelines/${id}`),
  createPipeline: (pipeline: CreatePipelineRequest) =>
    request<CreatePipelineResponse>('/api/pipelines', {
      method: 'POST',
      body: JSON.stringify(pipeline),
    }),
  createAndRunPipeline: (pipeline: CreatePipelineRequest) =>
    request<CreateAndRunPipelineResponse>('/api/pipelines/run', {
      method: 'POST',
      body: JSON.stringify(pipeline),
    }),
  runPipeline: (id: string) => request<RunPipelineResponse>(`/api/pipelines/${id}/run`, { method: 'POST' }),
  cancelPipeline: (id: string) => request<{ status: string }>(`/api/pipelines/${id}/cancel`, { method: 'POST' }),
  deletePipeline: (id: string) => request<{ status: string }>(`/api/pipelines/${id}`, { method: 'DELETE' }),

  // Runs
  listRuns: (pipelineId?: string) =>
    request<PipelineRun[]>(pipelineId ? `/api/runs?pipeline_id=${pipelineId}` : '/api/runs'),
  getRun: (id: string) => request<PipelineRun>(`/api/runs/${id}`),
  cancelRun: (id: string) => request<{ status: string }>(`/api/runs/${id}/cancel`, { method: 'POST' }),
  deleteRun: (id: string) => request<{ status: string }>(`/api/runs/${id}`, { method: 'DELETE' }),

  // AI layer (Laya)
  aiStatus: () => request<AIStatus>('/api/ai/status'),
  aiDecisions: (limit = 50) => request<Decision[]>(`/api/ai/decisions?limit=${limit}`),
  aiNotifications: () => request<Notification[]>('/api/ai/notifications'),
  aiTriage: (logs: string, source?: string) =>
    request<TriageResult>('/api/ai/triage', {
      method: 'POST',
      body: JSON.stringify({ logs, source }),
    }),
  aiRouteIssue: (title: string, body = '', labels: string[] = []) =>
    request<IssueRouteResult>('/api/ai/issue', {
      method: 'POST',
      body: JSON.stringify({ title, body, labels }),
    }),
  aiSuggestFix: (pipeline: string, step: PipelineStep) =>
    request<FixSuggestion>('/api/ai/fix', {
      method: 'POST',
      body: JSON.stringify({ pipeline, step }),
    }),
  aiFeedback: (feedback: FeedbackPayload) =>
    request<{ saved: boolean }>('/api/ai/feedback', {
      method: 'POST',
      body: JSON.stringify(feedback),
    }),
  listFeedback: (limit = 50) => request<FeedbackEntry[]>(`/api/ai/feedback?limit=${limit}`),

  // Logs & Artifacts
  listArtifacts: (runId: string) => request<ArtifactInfo[]>(`/api/runs/${runId}/artifacts`),
  getArtifactUrl: (runId: string, step: string, name: string) =>
    `/api/runs/${runId}/artifacts/${encodeURIComponent(step)}/${encodeURIComponent(name)}`,
  getRunLogs: (runId: string) =>
    request<LogEvent[]>(`/api/runs/${runId}/logs`, {
      headers: { Accept: 'application/json' },
    }),
};
