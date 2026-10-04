'use client';

import Link from 'next/link';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import {
  AlertTriangle,
  ArrowLeft,
  BrainCircuit,
  Check,
  ChevronDown,
  Clipboard,
  Code2,
  Download,
  FolderArchive,
  LoaderCircle,
  Play,
  Radio,
  RefreshCw,
  RotateCcw,
  Sparkles,
  Square,
  Terminal,
  ThumbsDown,
  ThumbsUp,
  Trash2,
} from 'lucide-react';

import { AppShell } from '@/components/app-shell';
import { StatusBadge } from '@/components/ui/status-badge';
import { apiClient } from '@/lib/api';
import { copyToClipboard, formatDuration, formatRelativeTime } from '@/lib/utils';
import type { ArtifactInfo, FixSuggestion, LogEvent, PipelineRun, PipelineStep } from '@/types/api';

export default function RunDetailsPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [run, setRun] = useState<PipelineRun | null>(null);
  const [artifacts, setArtifacts] = useState<ArtifactInfo[]>([]);
  const [liveLogs, setLiveLogs] = useState<LogEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedSteps, setExpandedSteps] = useState<Set<number>>(new Set());
  const [copied, setCopied] = useState(false);
  const [showConsole, setShowConsole] = useState(false);

  const loadData = useCallback(async () => {
    try {
      const data = await apiClient.getRun(id);
      setRun(data);
      setExpandedSteps((current) => {
        if (current.size) return current;
        const important = new Set<number>();
        data.steps?.forEach((step, index) => {
          if (step.status === 'failed' || step.status === 'running') important.add(index);
        });
        return important;
      });
      setError(null);

      // Load artifacts
      try {
        const arts = await apiClient.listArtifacts(id);
        setArtifacts(arts || []);
      } catch {
        // quiet
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to load run.');
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    loadData();
    const interval = window.setInterval(loadData, 4000);
    return () => window.clearInterval(interval);
  }, [loadData]);

  // Real-time SSE log connection
  useEffect(() => {
    if (!id) return;
    const es = new EventSource(`/api/runs/${id}/logs`);
    es.onmessage = (event) => {
      try {
        const data: LogEvent = JSON.parse(event.data);
        setLiveLogs((prev) => {
          if (prev.some((l) => l.timestamp === data.timestamp && l.line === data.line && l.step_name === data.step_name)) {
            return prev;
          }
          return [...prev, data];
        });
      } catch {
        // ping or raw line
      }
    };
    es.onerror = () => {
      es.close();
    };
    return () => {
      es.close();
    };
  }, [id]);

  const duration = run?.total_duration ?? elapsedSeconds(run?.started_at);
  const completedSteps = run?.steps?.filter((step) => step.status === 'success' || step.status === 'failed' || step.status === 'skipped').length ?? 0;
  const stepCount = run?.steps?.length ?? 0;
  const progress = stepCount ? Math.round((completedSteps / stepCount) * 100) : 0;

  const toggleStep = (index: number) => {
    setExpandedSteps((current) => {
      const next = new Set(current);
      if (next.has(index)) next.delete(index); else next.add(index);
      return next;
    });
  };

  const rerun = async () => {
    if (!run) return;
    try {
      const result = await apiClient.runPipeline(run.pipeline_id);
      router.push(`/run/${result.run_id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to rerun pipeline.');
    }
  };

  const cancel = async () => {
    if (!run) return;
    try {
      await apiClient.cancelRun(run.id);
      await loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to cancel run.');
    }
  };

  const remove = async () => {
    if (!run || !window.confirm(`Delete run “${run.id.slice(0, 8)}”?`)) return;
    try {
      await apiClient.deleteRun(run.id);
      router.push(`/pipeline/${run.pipeline_id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to delete run.');
    }
  };

  const copyId = async () => {
    if (!run) return;
    await copyToClipboard(run.id);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1800);
  };

  if (loading) {
    return (
      <AppShell eyebrow="Execution trace" title="Loading run" description="Reading current engine state and step telemetry.">
        <div className="panel grid min-h-80 place-items-center"><LoaderCircle className="size-7 animate-spin text-[var(--signal)]" /></div>
      </AppShell>
    );
  }

  if (!run) {
    return (
      <AppShell eyebrow="Execution trace" title="Run unavailable" description="The engine could not return this execution record.">
        <div className="panel p-6 text-sm text-red-200">{error ?? 'Run not found.'}</div>
      </AppShell>
    );
  }

  return (
    <AppShell
      eyebrow={`Run / ${run.id.slice(0, 8)}`}
      title={run.name}
      description="Step-by-step execution trace with failure intelligence and captured command output."
      actions={
        <>
          <Link className="btn-secondary" href={`/pipeline/${run.pipeline_id}`}><ArrowLeft className="size-4" />Pipeline</Link>
          <button className="btn-secondary" onClick={rerun}><RotateCcw className="size-4" />Rerun</button>
          {run.status === 'running' && <button className="btn-danger" onClick={cancel}><Square className="size-3.5" />Cancel</button>}
        </>
      }
    >
      {error && <div className="mb-5 flex items-center gap-3 border border-[var(--danger)]/25 bg-[var(--danger)]/6 px-4 py-3 text-sm text-red-200"><AlertTriangle className="size-4" />{error}</div>}

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4" aria-label="Run summary">
        <Summary label="Status"><StatusBadge status={run.status} /></Summary>
        <Summary label="Elapsed" value={duration != null ? formatDuration(duration) : '—'} detail={run.status === 'running' ? 'still running' : 'total duration'} />
        <Summary label="Progress" value={`${progress}%`} detail={`${completedSteps} of ${stepCount} steps`} />
        <Summary label="Started" value={run.started_at ? formatRelativeTime(run.started_at) : 'Queued'} detail={run.started_at ? new Date(run.started_at).toLocaleTimeString() : 'awaiting executor'} />
      </section>

      <div className="mt-6 grid gap-6 xl:grid-cols-[minmax(0,1fr)_20rem]">
        <section className="panel min-w-0">
          <div className="panel-header">
            <div className="flex items-center gap-2.5">
              <Terminal className="size-4 text-[var(--signal)]" />
              <h2 className="panel-title">Execution timeline</h2>
            </div>
            <div className="flex items-center gap-2">
              <button
                className={`btn-ghost text-xs ${showConsole ? 'border-[var(--cyan)]/30 text-[var(--cyan)]' : ''}`}
                onClick={() => setShowConsole((prev) => !prev)}
              >
                <Radio className="size-3.5" />
                Live Console ({liveLogs.length})
              </button>
              <button className="btn-ghost text-xs" onClick={loadData}><RefreshCw className="size-3.5" />Refresh</button>
            </div>
          </div>

          {showConsole && (
            <div className="border-b border-white/8 bg-[#060807] p-3">
              <div className="mb-2 flex items-center justify-between font-mono text-[10px] text-white/40">
                <span>REAL-TIME STREAMING BUFFER</span>
                <button className="text-[var(--cyan)] hover:underline" onClick={() => setLiveLogs([])}>Clear</button>
              </div>
              <div className="max-h-64 overflow-y-auto space-y-1 font-mono text-[11px] leading-4 text-white/70">
                {liveLogs.length === 0 ? (
                  <p className="text-white/30 italic">No real-time logs received yet.</p>
                ) : (
                  liveLogs.map((log, i) => (
                    <div key={i} className="flex gap-2">
                      <span className="shrink-0 text-white/25">{log.timestamp ? new Date(log.timestamp).toLocaleTimeString() : ''}</span>
                      <span className={`shrink-0 font-semibold ${log.stream === 'system' ? 'text-[var(--signal)]' : log.stream === 'stderr' ? 'text-red-400' : 'text-[var(--cyan)]'}`}>
                        [{log.step_name || 'engine'}]
                      </span>
                      <span className="break-all">{log.line}</span>
                    </div>
                  ))
                )}
              </div>
            </div>
          )}

          {!run.steps?.length ? (
            <div className="grid min-h-56 place-items-center text-sm text-[var(--muted)]">No steps recorded.</div>
          ) : (
            <div className="p-4 sm:p-5">
              {run.steps.map((step, index) => (
                <StepTrace
                  key={`${step.name}-${index}`}
                  step={step}
                  runName={run.name}
                  runId={run.id}
                  index={index}
                  isLast={index === run.steps!.length - 1}
                  expanded={expandedSteps.has(index)}
                  onToggle={() => toggleStep(index)}
                />
              ))}
            </div>
          )}
        </section>

        <aside className="space-y-4">
          <section className="panel p-4">
            <p className="panel-title">Run identity</p>
            <button className="mt-4 flex w-full items-center justify-between border border-white/8 bg-black/15 p-3 text-left" onClick={copyId}>
              <span className="min-w-0">
                <span className="block font-mono text-[10px] uppercase tracking-[0.12em] text-white/30">Run ID</span>
                <span className="mt-1 block truncate font-mono text-xs text-white/70">{run.id}</span>
              </span>
              {copied ? <Check className="size-4 shrink-0 text-[var(--signal)]" /> : <Clipboard className="size-4 shrink-0 text-white/30" />}
            </button>
            <div className="mt-2 border border-white/8 bg-black/15 p-3">
              <p className="font-mono text-[10px] uppercase tracking-[0.12em] text-white/30">Pipeline ID</p>
              <p className="mt-1 truncate font-mono text-xs text-white/70">{run.pipeline_id}</p>
            </div>
          </section>

          {/* Captured Artifacts Section */}
          <section className="panel p-4">
            <div className="flex items-center gap-2">
              <FolderArchive className="size-4 text-[var(--cyan)]" />
              <p className="panel-title">Artifacts ({artifacts.length})</p>
            </div>
            {artifacts.length === 0 ? (
              <p className="mt-3 text-xs text-white/40">No build artifacts captured for this run.</p>
            ) : (
              <div className="mt-3 space-y-2">
                {artifacts.map((art, idx) => (
                  <div key={idx} className="flex items-center justify-between border border-white/8 bg-black/20 p-2.5">
                    <div className="min-w-0 pr-2">
                      <p className="truncate font-mono text-xs text-white">{art.name}</p>
                      <p className="font-mono text-[9px] text-white/35">
                        {art.step_name} · {(art.size / 1024).toFixed(1)} KB
                      </p>
                    </div>
                    <a
                      href={`/api/runs/${run.id}/artifacts/${encodeURIComponent(art.step_name)}/${encodeURIComponent(art.name)}`}
                      download={art.name}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="btn-ghost p-1 text-[var(--cyan)]"
                      title="Download artifact"
                    >
                      <Download className="size-3.5" />
                    </a>
                  </div>
                ))}
              </div>
            )}
          </section>

          <section className="panel p-4">
            <p className="panel-title">Run controls</p>
            <div className="mt-4 grid gap-2">
              <button className="btn-secondary w-full" onClick={rerun}><Play className="size-4" />Start new run</button>
              <button className="btn-danger w-full" onClick={remove} disabled={run.status === 'running'}><Trash2 className="size-4" />Delete record</button>
            </div>
          </section>
        </aside>
      </div>
    </AppShell>
  );
}

function StepTrace({
  step,
  runName,
  runId,
  index,
  isLast,
  expanded,
  onToggle,
}: {
  step: PipelineStep;
  runName: string;
  runId: string;
  index: number;
  isLast: boolean;
  expanded: boolean;
  onToggle: () => void;
}) {
  const [fixSuggestion, setFixSuggestion] = useState<FixSuggestion | null>(null);
  const [loadingFix, setLoadingFix] = useState(false);
  const [copiedFix, setCopiedFix] = useState(false);

  const duration = useMemo(() => {
    if (!step.start_time || !step.end_time) return null;
    return Math.max(0, (new Date(step.end_time).getTime() - new Date(step.start_time).getTime()) / 1000);
  }, [step.start_time, step.end_time]);

  const requestFix = async () => {
    setLoadingFix(true);
    try {
      const suggestion = await apiClient.aiSuggestFix(runName, step);
      setFixSuggestion(suggestion);
    } catch {
      // ignore
    } finally {
      setLoadingFix(false);
    }
  };

  const copyFixCmd = async () => {
    if (!fixSuggestion?.suggested_command) return;
    await copyToClipboard(fixSuggestion.suggested_command);
    setCopiedFix(true);
    setTimeout(() => setCopiedFix(false), 2000);
  };

  return (
    <article className="relative grid grid-cols-[2.4rem_minmax(0,1fr)] gap-3">
      {!isLast && <span className="absolute bottom-0 left-[1.17rem] top-9 w-px bg-white/10" />}
      <span className={`relative z-10 grid size-9 place-items-center border ${stepTone(step.status)}`}>
        {step.status === 'running' ? (
          <LoaderCircle className="size-4 animate-spin" />
        ) : step.status === 'success' ? (
          <Check className="size-4" />
        ) : step.status === 'failed' ? (
          <AlertTriangle className="size-4" />
        ) : (
          <span className="font-mono text-[10px]">{String(index + 1).padStart(2, '0')}</span>
        )}
      </span>
      <div className={`mb-4 border ${step.status === 'failed' ? 'border-[var(--danger)]/22 bg-[var(--danger)]/[0.025]' : 'border-white/9 bg-black/10'}`}>
        <button className="grid w-full grid-cols-[minmax(0,1fr)_auto] items-center gap-3 p-3.5 text-left" onClick={onToggle} aria-expanded={expanded}>
          <span className="min-w-0">
            <span className="flex flex-wrap items-center gap-2">
              <span className="truncate text-sm font-medium text-white">{step.name}</span>
              {step.status && <StatusBadge status={step.status} />}
              {step.depends_on && step.depends_on.length > 0 && (
                <span className="border border-[var(--cyan)]/25 bg-[var(--cyan)]/8 px-1.5 py-0.5 font-mono text-[9px] text-[var(--cyan)]">
                  DAG: {step.depends_on.join(', ')}
                </span>
              )}
              {step.artifacts && step.artifacts.length > 0 && (
                <span className="border border-white/15 bg-white/5 px-1.5 py-0.5 font-mono text-[9px] text-white/50">
                  artifacts: {step.artifacts.length}
                </span>
              )}
            </span>
            <span className="mt-1.5 block truncate font-mono text-[10px] text-white/38">{step.command}</span>
          </span>
          <span className="flex items-center gap-3">
            {duration != null && <span className="font-mono text-[10px] text-white/35">{formatDuration(duration)}</span>}
            <ChevronDown className={`size-4 text-white/30 transition ${expanded ? 'rotate-180' : ''}`} />
          </span>
        </button>

        {expanded && (
          <div className="border-t border-white/8 p-3.5 fade-up">
            {step.ai_analysis && <AIAnalysisPanel step={step} runId={runId} />}

            {step.status === 'failed' && (
              <div className="mb-3">
                {!fixSuggestion ? (
                  <button
                    className="btn-secondary flex items-center gap-1.5 text-xs text-[var(--cyan)]"
                    onClick={requestFix}
                    disabled={loadingFix}
                  >
                    {loadingFix ? <LoaderCircle className="size-3.5 animate-spin" /> : <Sparkles className="size-3.5" />}
                    {loadingFix ? 'Diagnosing...' : 'AI Suggest Fix'}
                  </button>
                ) : (
                  <div className="border border-[var(--cyan)]/20 bg-[var(--cyan)]/5 p-3">
                    <div className="flex items-center gap-1.5 font-mono text-[10px] font-semibold uppercase tracking-[0.12em] text-[var(--cyan)]">
                      <Sparkles className="size-3.5" />
                      Suggested Fix Hypothesis
                    </div>
                    <p className="mt-1 text-xs text-white/80">{fixSuggestion.hypothesis}</p>
                    {fixSuggestion.suggested_command && (
                      <div className="mt-2.5 flex items-center justify-between border border-white/10 bg-black/40 p-2 font-mono text-xs text-[var(--signal)]">
                        <span className="truncate pr-2">{fixSuggestion.suggested_command}</span>
                        <button className="btn-ghost p-1" onClick={copyFixCmd} title="Copy command">
                          {copiedFix ? <Check className="size-3 text-[var(--signal)]" /> : <Clipboard className="size-3 text-white/40" />}
                        </button>
                      </div>
                    )}
                  </div>
                )}
              </div>
            )}

            <div className="mt-3 grid gap-2 sm:grid-cols-3">
              <Meta label="Timeout" value={`${step.timeout || 300}s`} />
              <Meta label="Retries" value={String(step.retry_count ?? 0)} />
              <Meta label="Continue on error" value={step.continue_on_error ? 'yes' : 'no'} />
            </div>

            {step.output && <LogBlock title="Command output" text={step.output} tone="normal" />}
            {step.error && <LogBlock title="Error output" text={step.error} tone="error" />}
          </div>
        )}
      </div>
    </article>
  );
}

function AIAnalysisPanel({ step, runId }: { step: PipelineStep; runId: string }) {
  const analysis = step.ai_analysis!;
  const retry = analysis.classification.retry ?? analysis.classification.retri;
  const [feedbackSent, setFeedbackSent] = useState(false);

  const sendFeedback = async (rating: 'positive' | 'negative') => {
    try {
      await apiClient.aiFeedback({
        rating,
        decision_id: `${step.name}-${runId}`,
        comment: `User rated verdict for step ${step.name}`,
      });
      setFeedbackSent(true);
    } catch {
      // quiet
    }
  };

  return (
    <section className="mb-4 border border-[var(--cyan)]/18 bg-[var(--cyan)]/[0.035]">
      <div className="flex items-center justify-between gap-3 border-b border-[var(--cyan)]/12 px-3 py-2.5">
        <div className="flex items-center gap-2 text-[var(--cyan)]">
          <BrainCircuit className="size-4" />
          <p className="font-mono text-[10px] font-semibold uppercase tracking-[0.14em]">Laya verdict</p>
        </div>
        <div className="flex items-center gap-2">
          {!feedbackSent ? (
            <span className="flex items-center gap-1 font-mono text-[9px] text-white/40">
              Helpful?
              <button
                className="hover:text-[var(--signal)] px-1 py-0.5"
                onClick={() => sendFeedback('positive')}
                title="Verdict was helpful"
              >
                <ThumbsUp className="size-3" />
              </button>
              <button
                className="hover:text-red-400 px-1 py-0.5"
                onClick={() => sendFeedback('negative')}
                title="Verdict was not helpful"
              >
                <ThumbsDown className="size-3" />
              </button>
            </span>
          ) : (
            <span className="font-mono text-[9px] text-[var(--signal)]">Thanks for feedback!</span>
          )}
        </div>
      </div>
      <div className="grid gap-3 p-3 sm:grid-cols-3">
        <Verdict label="Failure kind" value={analysis.classification.kind} />
        <Verdict label="Recommended action" value={analysis.log_triage.action} />
        <Verdict label="Retry" value={retry ? 'recommended' : 'not recommended'} accent={retry} />
      </div>
      <div className="grid gap-px border-t border-[var(--cyan)]/12 bg-[var(--cyan)]/12 sm:grid-cols-3">
        <Confidence label="Retry confidence" value={analysis.classification.retry_probability} />
        <Confidence label="Page probability" value={analysis.log_triage.page_probability} />
        <Confidence
          label="Severity"
          value={analysis.classification.severity != null ? analysis.classification.severity / 3 : null}
          display={analysis.classification.severity?.toFixed(1)}
        />
      </div>
    </section>
  );
}

function Confidence({ label, value, display }: { label: string; value?: number | null; display?: string }) {
  const normalized = value == null ? 0 : Math.min(1, Math.max(0, value));
  return (
    <div className="bg-[#0d1716] p-3">
      <div className="flex items-center justify-between gap-2 font-mono text-[9px] uppercase tracking-[0.08em] text-white/35">
        <span>{label}</span>
        <span className="text-white/60">{display ?? (value == null ? '—' : `${Math.round(value * 100)}%`)}</span>
      </div>
      <div className="mt-2 h-1 bg-white/8">
        <div className="h-full bg-[var(--cyan)]" style={{ width: `${Math.max(normalized * 100, value == null ? 0 : 2)}%` }} />
      </div>
    </div>
  );
}

function Verdict({ label, value, accent = false }: { label: string; value: string; accent?: boolean }) {
  return (
    <div>
      <p className="font-mono text-[9px] uppercase tracking-[0.11em] text-white/30">{label}</p>
      <p className={`mt-1.5 text-sm font-medium capitalize ${accent ? 'text-[var(--signal)]' : 'text-white'}`}>{value}</p>
    </div>
  );
}

function LogBlock({ title, text, tone }: { title: string; text: string; tone: 'normal' | 'error' }) {
  return (
    <details className={`mt-3 border ${tone === 'error' ? 'border-[var(--danger)]/18' : 'border-white/8'}`} open={tone === 'error'}>
      <summary className={`flex cursor-pointer items-center gap-2 px-3 py-2.5 font-mono text-[10px] uppercase tracking-[0.11em] ${tone === 'error' ? 'text-[var(--danger)]' : 'text-white/45'}`}>
        <Code2 className="size-3.5" />
        {title}
      </summary>
      <pre className="max-h-80 overflow-auto border-t border-white/8 bg-[#070a09] p-3 font-mono text-[11px] leading-5 text-white/62">{text}</pre>
    </details>
  );
}

function Summary({ label, value, detail, children }: { label: string; value?: string; detail?: string; children?: React.ReactNode }) {
  return (
    <article className="metric">
      <p className="font-mono text-[10px] uppercase tracking-[0.16em] text-white/42">{label}</p>
      <div className="mt-4 min-h-9">{children ?? <p className="text-2xl font-semibold tracking-[-0.03em] text-white">{value}</p>}</div>
      {detail && <p className="mt-1 font-mono text-[10px] uppercase tracking-[0.09em] text-white/35">{detail}</p>}
    </article>
  );
}

function Meta({ label, value }: { label: string; value: string }) {
  return (
    <div className="border border-white/8 bg-black/15 p-2.5">
      <p className="font-mono text-[9px] uppercase tracking-[0.1em] text-white/28">{label}</p>
      <p className="mt-1 text-xs text-white/65">{value}</p>
    </div>
  );
}

function stepTone(status?: string): string {
  if (status === 'success') return 'border-[var(--signal)]/24 bg-[var(--signal)]/7 text-[var(--signal)]';
  if (status === 'failed') return 'border-[var(--danger)]/24 bg-[var(--danger)]/7 text-[var(--danger)]';
  if (status === 'running') return 'border-[var(--cyan)]/24 bg-[var(--cyan)]/7 text-[var(--cyan)]';
  return 'border-white/12 bg-white/3 text-white/35';
}

function elapsedSeconds(startedAt?: string): number | null {
  if (!startedAt) return null;
  return Math.max(0, (Date.now() - new Date(startedAt).getTime()) / 1000);
}
