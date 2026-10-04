'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  Activity,
  AlertTriangle,
  ArrowRight,
  BrainCircuit,
  CheckCircle2,
  ChevronDown,
  CircleOff,
  Clock3,
  GitBranch,
  LoaderCircle,
  Play,
  Plus,
  RefreshCw,
  Square,
  Trash2,
  X,
} from 'lucide-react';

import { AppShell } from '@/components/app-shell';
import { StatusBadge } from '@/components/ui/status-badge';
import { apiClient } from '@/lib/api';
import { defaultPipelineTemplate, formatDuration, formatRelativeTime } from '@/lib/utils';
import type { AIStatus, CreatePipelineRequest, Pipeline, PipelineRun } from '@/types/api';

export default function Dashboard() {
  const router = useRouter();
  const [pipelines, setPipelines] = useState<Pipeline[]>([]);
  const [runs, setRuns] = useState<PipelineRun[]>([]);
  const [aiStatus, setAIStatus] = useState<AIStatus | null>(null);
  const [pipelineJson, setPipelineJson] = useState(JSON.stringify(defaultPipelineTemplate, null, 2));
  const [editorOpen, setEditorOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [refreshing, setRefreshing] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    const [pipelineResult, runResult, aiResult] = await Promise.allSettled([
      apiClient.listPipelines(),
      apiClient.listRuns(),
      apiClient.aiStatus(),
    ]);

    if (pipelineResult.status === 'fulfilled') setPipelines(pipelineResult.value);
    if (runResult.status === 'fulfilled') setRuns(runResult.value);
    if (aiResult.status === 'fulfilled') setAIStatus(aiResult.value);

    const failed = [pipelineResult, runResult].filter((result) => result.status === 'rejected');
    setError(failed.length ? 'Backend connection interrupted. Showing the last successful snapshot.' : null);
    setRefreshing(false);
  }, []);

  useEffect(() => {
    loadData();
    const interval = window.setInterval(loadData, 5000);
    return () => window.clearInterval(interval);
  }, [loadData]);

  const metrics = useMemo(() => {
    const running = runs.filter((run) => run.status === 'running').length;
    const failed = runs.filter((run) => run.status === 'failed').length;
    const completed = runs.filter((run) => run.status === 'success' || run.status === 'failed');
    const success = completed.filter((run) => run.status === 'success').length;
    return {
      running,
      failed,
      successRate: completed.length ? Math.round((success / completed.length) * 100) : 0,
    };
  }, [runs]);

  const failedRuns = runs.filter((run) => run.status === 'failed').slice(0, 5);
  const recentRuns = runs.slice(0, 7);

  const flash = (message: string) => {
    setNotice(message);
    window.setTimeout(() => setNotice(null), 3500);
  };

  const parsePipeline = (): CreatePipelineRequest => {
    const parsed = JSON.parse(pipelineJson) as CreatePipelineRequest;
    if (!parsed.name || !Array.isArray(parsed.steps) || parsed.steps.length === 0) {
      throw new Error('Pipeline name and at least one step are required.');
    }
    return parsed;
  };

  const createPipeline = async (runNow: boolean) => {
    try {
      setLoading(true);
      setError(null);
      const pipeline = parsePipeline();
      if (runNow) {
        const result = await apiClient.createAndRunPipeline(pipeline);
        flash(`Run ${result.run_id.slice(0, 8)} started.`);
        router.push(`/run/${result.run_id}`);
      } else {
        await apiClient.createPipeline(pipeline);
        flash('Pipeline saved.');
        setEditorOpen(false);
        await loadData();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to create pipeline.');
    } finally {
      setLoading(false);
    }
  };

  const runPipeline = async (id: string) => {
    try {
      const result = await apiClient.runPipeline(id);
      router.push(`/run/${result.run_id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to start pipeline.');
    }
  };

  const cancelRun = async (id: string) => {
    try {
      await apiClient.cancelRun(id);
      flash('Cancellation requested.');
      await loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to cancel run.');
    }
  };

  const deletePipeline = async (pipeline: Pipeline) => {
    if (!window.confirm(`Delete pipeline “${pipeline.name}”?`)) return;
    try {
      await apiClient.deletePipeline(pipeline.id);
      flash('Pipeline deleted.');
      await loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to delete pipeline.');
    }
  };

  return (
    <AppShell
      eyebrow="Control plane / live"
      title="Pipeline operations"
      description="See what is running, what needs attention, and what Laya decided—without digging through raw execution data."
      actions={
        <>
          <div className="flex min-h-10 items-center gap-2 border border-white/10 bg-white/3 px-3 font-mono text-[10px] uppercase tracking-[0.14em] text-white/55">
            <span className={`signal-dot ${error ? 'text-[var(--danger)]' : 'text-[var(--signal)]'}`} />
            {error ? 'Snapshot mode' : 'Backend live'}
          </div>
          <button className="btn-primary" onClick={() => setEditorOpen((open) => !open)}>
            {editorOpen ? <X className="size-4" /> : <Plus className="size-4" />}
            {editorOpen ? 'Close editor' : 'New pipeline'}
          </button>
        </>
      }
    >
      {(error || notice) && (
        <div className={`mb-5 flex items-center gap-3 border px-4 py-3 text-sm ${error ? 'border-[var(--danger)]/30 bg-[var(--danger)]/6 text-red-200' : 'border-[var(--signal)]/25 bg-[var(--signal)]/6 text-[var(--signal)]'}`}>
          {error ? <CircleOff className="size-4 shrink-0" /> : <CheckCircle2 className="size-4 shrink-0" />}
          <span>{error ?? notice}</span>
        </div>
      )}

      {editorOpen && (
        <section className="panel mb-6 fade-up" aria-labelledby="pipeline-editor-title">
          <div className="panel-header">
            <div>
              <p className="panel-title" id="pipeline-editor-title">Pipeline launchpad</p>
              <p className="mt-1 text-xs text-[var(--muted)]">JSON contract · validated before launch</p>
            </div>
            <span className="font-mono text-[10px] text-white/35">CTRL PLANE INPUT</span>
          </div>
          <div className="grid gap-5 p-4 lg:grid-cols-[minmax(0,1fr)_17rem] lg:p-5">
            <textarea
              value={pipelineJson}
              onChange={(event) => setPipelineJson(event.target.value)}
              className="field min-h-80 resize-y font-mono leading-6"
              aria-label="Pipeline JSON"
              spellCheck={false}
            />
            <div className="flex flex-col justify-between gap-6 border border-white/8 bg-black/15 p-4">
              <div>
                <p className="font-mono text-[10px] uppercase tracking-[0.16em] text-[var(--signal)]">Launch checklist</p>
                <ol className="mt-4 space-y-3 text-xs leading-5 text-[var(--muted)]">
                  <li className="flex gap-3"><span className="text-white/35">01</span> Name the pipeline and version the contract.</li>
                  <li className="flex gap-3"><span className="text-white/35">02</span> Add commands, timeouts, and retry limits.</li>
                  <li className="flex gap-3"><span className="text-white/35">03</span> Laya will classify failures automatically.</li>
                </ol>
              </div>
              <div className="grid gap-2">
                <button className="btn-primary" disabled={loading} onClick={() => createPipeline(true)}>
                  {loading ? <LoaderCircle className="size-4 animate-spin" /> : <Play className="size-4" />}
                  Create & run
                </button>
                <button className="btn-secondary" disabled={loading} onClick={() => createPipeline(false)}>
                  <GitBranch className="size-4" /> Save definition
                </button>
              </div>
            </div>
          </div>
        </section>
      )}

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4" aria-label="Operational summary">
        <Metric label="Active runs" value={metrics.running} detail="executing now" icon={<Activity className="size-4" />} tone="var(--cyan)" />
        <Metric label="Failed runs" value={metrics.failed} detail="in retained history" icon={<AlertTriangle className="size-4" />} tone="var(--danger)" />
        <Metric label="Success rate" value={`${metrics.successRate}%`} detail="completed runs" icon={<CheckCircle2 className="size-4" />} tone="var(--signal)" />
        <Metric
          label="Decision layer"
          value={aiStatus?.configured ? 'LIVE' : 'SAFE'}
          detail={aiStatus?.configured ? 'Laya gateway' : 'heuristic fallback'}
          icon={<BrainCircuit className="size-4" />}
          tone={aiStatus?.configured ? 'var(--signal)' : 'var(--amber)'}
        />
      </section>

      <div className="mt-6 grid gap-6 xl:grid-cols-[0.82fr_1.18fr]">
        <section className="panel min-w-0">
          <div className="panel-header">
            <div className="flex items-center gap-2.5">
              <AlertTriangle className="size-4 text-[var(--danger)]" />
              <h2 className="panel-title">Needs attention</h2>
            </div>
            <span className="font-mono text-[10px] text-white/35">{failedRuns.length} OPEN</span>
          </div>
          {failedRuns.length === 0 ? (
            <div className="grid min-h-64 place-items-center p-8 text-center">
              <div>
                <CheckCircle2 className="mx-auto size-8 text-[var(--signal)]" strokeWidth={1.4} />
                <p className="mt-4 text-sm font-medium text-white">No failed runs</p>
                <p className="mt-1 text-xs text-[var(--muted)]">The queue is clear.</p>
              </div>
            </div>
          ) : (
            <div>
              {failedRuns.map((run) => (
                <button key={run.id} className="data-row w-full grid-cols-[auto_minmax(0,1fr)_auto] text-left" onClick={() => router.push(`/run/${run.id}`)}>
                  <span className="grid size-9 place-items-center border border-[var(--danger)]/20 bg-[var(--danger)]/7 text-[var(--danger)]"><AlertTriangle className="size-4" /></span>
                  <span className="min-w-0">
                    <span className="block truncate text-sm font-medium text-white">{run.name}</span>
                    <span className="mt-1 block font-mono text-[10px] text-white/38">{run.id.slice(0, 8)} · {formatRelativeTime(run.created_at)}</span>
                  </span>
                  <ArrowRight className="size-4 text-white/30" />
                </button>
              ))}
            </div>
          )}
        </section>

        <section className="panel min-w-0">
          <div className="panel-header">
            <div className="flex items-center gap-2.5">
              <Activity className="size-4 text-[var(--cyan)]" />
              <h2 className="panel-title">Recent execution</h2>
            </div>
            <button className="btn-ghost" onClick={loadData} disabled={refreshing} aria-label="Refresh runs">
              <RefreshCw className={`size-3.5 ${refreshing ? 'animate-spin' : ''}`} /> Refresh
            </button>
          </div>
          {recentRuns.length === 0 ? (
            <Empty label="No runs recorded" />
          ) : (
            <div>
              {recentRuns.map((run) => (
                <div key={run.id} className="data-row grid-cols-[minmax(0,1fr)_auto_auto]">
                  <button className="min-w-0 text-left" onClick={() => router.push(`/run/${run.id}`)}>
                    <span className="block truncate text-sm font-medium text-white hover:text-[var(--signal)]">{run.name}</span>
                    <span className="mt-1 block font-mono text-[10px] text-white/38">{formatRelativeTime(run.created_at)} · {run.id.slice(0, 8)}</span>
                  </button>
                  <span className="hidden font-mono text-[10px] text-white/45 sm:block">{run.total_duration != null ? formatDuration(run.total_duration) : '—'}</span>
                  <div className="flex items-center gap-1">
                    <StatusBadge status={run.status} />
                    {run.status === 'running' && (
                      <button className="btn-ghost px-2" onClick={() => cancelRun(run.id)} aria-label={`Cancel ${run.name}`}><Square className="size-3.5" /></button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </section>
      </div>

      <section className="panel mt-6 min-w-0">
        <div className="panel-header">
          <div className="flex items-center gap-2.5">
            <GitBranch className="size-4 text-[var(--signal)]" />
            <h2 className="panel-title">Pipeline inventory</h2>
          </div>
          <span className="font-mono text-[10px] text-white/35">{pipelines.length} DEFINITIONS</span>
        </div>
        {pipelines.length === 0 ? (
          <Empty label="No pipeline definitions" action={<button className="btn-secondary mt-4" onClick={() => setEditorOpen(true)}><Plus className="size-4" />Create one</button>} />
        ) : (
          <div>
            {pipelines.map((pipeline) => (
              <div key={pipeline.id} className="data-row grid-cols-[minmax(0,1.2fr)_minmax(0,.8fr)_auto] lg:grid-cols-[minmax(0,1.2fr)_8rem_7rem_8rem_auto]">
                <button className="min-w-0 text-left" onClick={() => router.push(`/pipeline/${pipeline.id}`)}>
                  <span className="block truncate text-sm font-medium text-white hover:text-[var(--signal)]">{pipeline.name}</span>
                  <span className="mt-1 block truncate text-xs text-[var(--muted)]">{pipeline.description || pipeline.id}</span>
                </button>
                <span className="hidden font-mono text-[10px] text-white/42 lg:block">{pipeline.total_runs ?? 0} runs</span>
                <span className="hidden font-mono text-[10px] text-white/42 lg:block">{formatRelativeTime(pipeline.last_run_at ?? pipeline.created_at)}</span>
                <StatusBadge status={pipeline.status} />
                <div className="flex justify-end gap-1">
                  <button className="btn-ghost px-2" onClick={() => runPipeline(pipeline.id)} aria-label={`Run ${pipeline.name}`}><Play className="size-3.5" /></button>
                  <button className="btn-ghost px-2 hover:text-[var(--danger)]" onClick={() => deletePipeline(pipeline)} aria-label={`Delete ${pipeline.name}`}><Trash2 className="size-3.5" /></button>
                  <button className="btn-ghost px-2" onClick={() => router.push(`/pipeline/${pipeline.id}`)} aria-label={`View ${pipeline.name}`}><ChevronDown className="size-3.5 -rotate-90" /></button>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>
    </AppShell>
  );
}

function Metric({ label, value, detail, icon, tone }: { label: string; value: string | number; detail: string; icon: React.ReactNode; tone: string }) {
  return (
    <article className="metric fade-up" style={{ color: tone }}>
      <div className="flex items-center justify-between">
        <p className="font-mono text-[10px] uppercase tracking-[0.16em] text-white/42">{label}</p>
        <span>{icon}</span>
      </div>
      <p className="mt-4 text-3xl font-semibold tracking-[-0.04em] text-white">{value}</p>
      <p className="mt-1 font-mono text-[10px] uppercase tracking-[0.11em] text-white/36">{detail}</p>
    </article>
  );
}

function Empty({ label, action }: { label: string; action?: React.ReactNode }) {
  return (
    <div className="grid min-h-44 place-items-center p-8 text-center">
      <div>
        <Clock3 className="mx-auto size-7 text-white/22" strokeWidth={1.4} />
        <p className="mt-3 text-sm text-[var(--muted)]">{label}</p>
        {action}
      </div>
    </div>
  );
}
