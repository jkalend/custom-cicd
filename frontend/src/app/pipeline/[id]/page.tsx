'use client';

import Link from 'next/link';
import { useCallback, useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import {
  AlertTriangle,
  ArrowLeft,
  ArrowRight,
  Braces,
  Clock3,
  Code2,
  GitBranch,
  LoaderCircle,
  Play,
  RefreshCw,
  Square,
  Trash2,
} from 'lucide-react';

import { AppShell } from '@/components/app-shell';
import { StatusBadge } from '@/components/ui/status-badge';
import { apiClient } from '@/lib/api';
import { formatDuration, formatRelativeTime } from '@/lib/utils';
import type { Pipeline, PipelineRun } from '@/types/api';

export default function PipelineDetailsPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [pipeline, setPipeline] = useState<Pipeline | null>(null);
  const [runs, setRuns] = useState<PipelineRun[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    const [pipelineResult, runsResult] = await Promise.allSettled([
      apiClient.getPipeline(id),
      apiClient.listRuns(id),
    ]);
    if (pipelineResult.status === 'fulfilled') setPipeline(pipelineResult.value);
    if (runsResult.status === 'fulfilled') setRuns(runsResult.value);
    setError(pipelineResult.status === 'rejected' || runsResult.status === 'rejected'
      ? 'Part of this pipeline snapshot could not be refreshed.'
      : null);
    setLoading(false);
  }, [id]);

  useEffect(() => {
    loadData();
    const interval = window.setInterval(loadData, 5000);
    return () => window.clearInterval(interval);
  }, [loadData]);

  const runPipeline = async () => {
    if (!pipeline) return;
    try {
      const result = await apiClient.runPipeline(pipeline.id);
      router.push(`/run/${result.run_id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to start pipeline.');
    }
  };

  const cancelPipeline = async () => {
    if (!pipeline) return;
    try {
      await apiClient.cancelPipeline(pipeline.id);
      await loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to cancel pipeline.');
    }
  };

  const deletePipeline = async () => {
    if (!pipeline || !window.confirm(`Delete pipeline “${pipeline.name}”?`)) return;
    try {
      await apiClient.deletePipeline(pipeline.id);
      router.push('/');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to delete pipeline.');
    }
  };

  if (loading) {
    return (
      <AppShell eyebrow="Pipeline definition" title="Loading pipeline" description="Reading definition and execution history.">
        <div className="panel grid min-h-80 place-items-center"><LoaderCircle className="size-7 animate-spin text-[var(--signal)]" /></div>
      </AppShell>
    );
  }

  if (!pipeline) {
    return (
      <AppShell eyebrow="Pipeline definition" title="Pipeline unavailable" description="The requested definition could not be loaded.">
        <div className="panel p-6 text-sm text-red-200">{error ?? 'Pipeline not found.'}</div>
      </AppShell>
    );
  }

  const failures = runs.filter((run) => run.status === 'failed').length;
  const active = runs.filter((run) => run.status === 'running').length;

  return (
    <AppShell
      eyebrow={`Pipeline / ${pipeline.id.slice(0, 8)}`}
      title={pipeline.name}
      description={pipeline.description || 'Versioned pipeline definition and its complete execution history.'}
      actions={
        <>
          <Link className="btn-secondary" href="/"><ArrowLeft className="size-4" />Operations</Link>
          <button className="btn-primary" onClick={runPipeline}><Play className="size-4" />Run pipeline</button>
          {active > 0 && <button className="btn-danger" onClick={cancelPipeline}><Square className="size-3.5" />Cancel active</button>}
        </>
      }
    >
      {error && <div className="mb-5 flex items-center gap-3 border border-[var(--amber)]/25 bg-[var(--amber)]/6 px-4 py-3 text-sm text-amber-100"><AlertTriangle className="size-4" />{error}</div>}

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4" aria-label="Pipeline summary">
        <Summary label="Current state"><StatusBadge status={pipeline.status} /></Summary>
        <Summary label="Version" value={pipeline.version || 'unversioned'} detail={`created ${formatRelativeTime(pipeline.created_at)}`} />
        <Summary label="Executions" value={String(runs.length)} detail={`${active} active · ${failures} failed`} />
        <Summary label="Last run" value={pipeline.last_run_at ? formatRelativeTime(pipeline.last_run_at) : 'Never'} detail={pipeline.last_run_at ? new Date(pipeline.last_run_at).toLocaleDateString() : 'no execution data'} />
      </section>

      <div className="mt-6 grid gap-6 xl:grid-cols-[minmax(0,1.2fr)_minmax(18rem,.8fr)]">
        <section className="panel min-w-0">
          <div className="panel-header">
            <div className="flex items-center gap-2.5"><Code2 className="size-4 text-[var(--signal)]" /><h2 className="panel-title">Step blueprint</h2></div>
            <span className="font-mono text-[10px] text-white/35">{pipeline.steps?.length ?? 0} STEPS</span>
          </div>
          {!pipeline.steps?.length ? (
            <Empty label="No steps in this definition" />
          ) : (
            <div className="p-4 sm:p-5">
              {pipeline.steps.map((step, index) => (
                <article key={`${step.name}-${index}`} className="relative grid grid-cols-[2.2rem_minmax(0,1fr)] gap-3">
                  {index !== pipeline.steps.length - 1 && <span className="absolute bottom-0 left-[1.05rem] top-8 w-px bg-white/10" />}
                  <span className="relative z-10 grid size-8 place-items-center border border-[var(--signal)]/18 bg-[var(--signal)]/6 font-mono text-[10px] text-[var(--signal)]">{String(index + 1).padStart(2, '0')}</span>
                  <div className="mb-3 border border-white/9 bg-black/12 p-3.5">
                    <div className="flex flex-wrap items-start justify-between gap-3">
                      <div>
                        <h3 className="text-sm font-medium text-white">{step.name}</h3>
                        {step.description && <p className="mt-1 text-xs text-[var(--muted)]">{step.description}</p>}
                        <div className="mt-2 flex flex-wrap items-center gap-2">
                          {step.depends_on && step.depends_on.length > 0 && (
                            <span className="border border-[var(--cyan)]/25 bg-[var(--cyan)]/8 px-1.5 py-0.5 font-mono text-[9px] text-[var(--cyan)]">
                              DAG depends on: {step.depends_on.join(', ')}
                            </span>
                          )}
                          {step.artifacts && step.artifacts.length > 0 && (
                            <span className="border border-white/15 bg-white/5 px-1.5 py-0.5 font-mono text-[9px] text-white/50">
                              artifacts: {step.artifacts.join(', ')}
                            </span>
                          )}
                        </div>
                      </div>
                      <span className="font-mono text-[9px] uppercase tracking-[0.1em] text-white/32">{step.timeout || 300}s · {step.retry_count ?? 0} retries</span>
                    </div>
                    <pre className="mt-3 overflow-x-auto border border-white/7 bg-[#080b0a] px-3 py-2.5 font-mono text-[11px] leading-5 text-white/58">{step.command}</pre>
                  </div>
                </article>
              ))}
            </div>
          )}
        </section>

        <aside className="space-y-6">
          <section className="panel">
            <div className="panel-header"><div className="flex items-center gap-2.5"><Braces className="size-4 text-[var(--cyan)]" /><h2 className="panel-title">Variables</h2></div></div>
            {pipeline.variables && Object.keys(pipeline.variables).length ? (
              <div>{Object.entries(pipeline.variables).map(([key, value]) => <div key={key} className="data-row grid-cols-[minmax(0,1fr)_auto]"><span className="truncate font-mono text-[10px] text-[var(--cyan)]">${key}</span><span className="max-w-40 truncate font-mono text-[10px] text-white/55">{String(value)}</span></div>)}</div>
            ) : <Empty label="No pipeline variables" compact />}
          </section>

          <section className="panel p-4">
            <p className="panel-title">Definition controls</p>
            <div className="mt-4 grid gap-2">
              <button className="btn-secondary w-full" onClick={loadData}><RefreshCw className="size-4" />Refresh snapshot</button>
              <button className="btn-danger w-full" onClick={deletePipeline} disabled={active > 0}><Trash2 className="size-4" />Delete pipeline</button>
            </div>
          </section>
        </aside>
      </div>

      <section className="panel mt-6 min-w-0">
        <div className="panel-header">
          <div className="flex items-center gap-2.5"><GitBranch className="size-4 text-[var(--cyan)]" /><h2 className="panel-title">Execution history</h2></div>
          <span className="font-mono text-[10px] text-white/35">{runs.length} RUNS</span>
        </div>
        {runs.length === 0 ? (
          <Empty label="No runs for this pipeline" action={<button className="btn-primary mt-4" onClick={runPipeline}><Play className="size-4" />Start first run</button>} />
        ) : (
          <div>
            {runs.slice(0, 20).map((run) => (
              <button key={run.id} className="data-row w-full grid-cols-[minmax(0,1fr)_auto_auto] text-left lg:grid-cols-[minmax(0,1fr)_8rem_8rem_auto]" onClick={() => router.push(`/run/${run.id}`)}>
                <span className="min-w-0"><span className="block truncate text-sm font-medium text-white hover:text-[var(--signal)]">{run.name}</span><span className="mt-1 block font-mono text-[10px] text-white/35">{run.id.slice(0, 8)}</span></span>
                <span className="hidden font-mono text-[10px] text-white/42 lg:block">{formatRelativeTime(run.created_at)}</span>
                <span className="hidden font-mono text-[10px] text-white/42 lg:block">{run.total_duration != null ? formatDuration(run.total_duration) : '—'}</span>
                <span className="flex items-center gap-3"><StatusBadge status={run.status} /><ArrowRight className="size-4 text-white/25" /></span>
              </button>
            ))}
          </div>
        )}
      </section>
    </AppShell>
  );
}

function Summary({ label, value, detail, children }: { label: string; value?: string; detail?: string; children?: React.ReactNode }) {
  return <article className="metric"><p className="font-mono text-[10px] uppercase tracking-[0.16em] text-white/42">{label}</p><div className="mt-4 min-h-9">{children ?? <p className="text-2xl font-semibold tracking-[-0.03em] text-white">{value}</p>}</div>{detail && <p className="mt-1 font-mono text-[10px] uppercase tracking-[0.09em] text-white/35">{detail}</p>}</article>;
}

function Empty({ label, compact = false, action }: { label: string; compact?: boolean; action?: React.ReactNode }) {
  return <div className={`grid place-items-center p-6 text-center ${compact ? 'min-h-28' : 'min-h-48'}`}><div><Clock3 className="mx-auto size-6 text-white/20" /><p className="mt-3 text-sm text-[var(--muted)]">{label}</p>{action}</div></div>;
}
