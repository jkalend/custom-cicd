'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  AlertTriangle,
  ArrowRight,
  Bell,
  BrainCircuit,
  CheckCircle2,
  FlaskConical,
  Gauge,
  LoaderCircle,
  Radio,
  RefreshCw,
  Route,
  Sparkles,
} from 'lucide-react';

import { AppShell } from '@/components/app-shell';
import { apiClient } from '@/lib/api';
import { formatRelativeTime } from '@/lib/utils';
import type { AIStatus, Decision, IssueRouteResult, Notification, TriageResult } from '@/types/api';

const moduleLabels: Record<string, string> = {
  classify_failure: 'Failure classification',
  triage_logs: 'Log triage',
  route_issue: 'Issue routing',
  filter_notification: 'Notification filter',
};

export default function AIDashboard() {
  const [status, setStatus] = useState<AIStatus | null>(null);
  const [decisions, setDecisions] = useState<Decision[]>([]);
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [selected, setSelected] = useState<Decision | null>(null);
  const [moduleFilter, setModuleFilter] = useState('all');
  const [error, setError] = useState<string | null>(null);

  const [triageInput, setTriageInput] = useState('ERROR connection to database reset after 3 retries\nOutOfMemoryError: Java heap space in worker-2');
  const [triageResult, setTriageResult] = useState<TriageResult | null>(null);
  const [triageLoading, setTriageLoading] = useState(false);

  const [issueTitle, setIssueTitle] = useState('After updating to 2.4, Windows crashes when opening settings');
  const [issueBody, setIssueBody] = useState('Crash happens only on first launch after the update.');
  const [issueResult, setIssueResult] = useState<IssueRouteResult | null>(null);
  const [issueLoading, setIssueLoading] = useState(false);

  const loadAll = useCallback(async () => {
    const [statusResult, decisionResult, notificationResult] = await Promise.allSettled([
      apiClient.aiStatus(),
      apiClient.aiDecisions(50),
      apiClient.aiNotifications(),
    ]);

    if (statusResult.status === 'fulfilled') setStatus(statusResult.value);
    if (decisionResult.status === 'fulfilled') {
      setDecisions(decisionResult.value);
      setSelected((current) => current ?? decisionResult.value[0] ?? null);
    }
    if (notificationResult.status === 'fulfilled') setNotifications(notificationResult.value);
    setError([statusResult, decisionResult, notificationResult].some((result) => result.status === 'rejected')
      ? 'Decision telemetry is temporarily unavailable. Showing the last successful snapshot.'
      : null);
  }, []);

  useEffect(() => {
    loadAll();
    const interval = window.setInterval(loadAll, 5000);
    return () => window.clearInterval(interval);
  }, [loadAll]);

  const modules = useMemo(() => Array.from(new Set(decisions.map((decision) => decision.context.module))), [decisions]);
  const visibleDecisions = moduleFilter === 'all' ? decisions : decisions.filter((decision) => decision.context.module === moduleFilter);
  const layaCount = decisions.filter((decision) => decision.provider === 'laya').length;
  const meanLatency = decisions.length ? Math.round(decisions.reduce((sum, decision) => sum + decision.latency_ms, 0) / decisions.length) : 0;
  const urgentCount = notifications.filter((notification) => notification.level === 'urgent').length;

  const runTriage = async () => {
    try {
      setTriageLoading(true);
      setTriageResult(await apiClient.aiTriage(triageInput, 'playground'));
      await loadAll();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Triage failed.');
    } finally {
      setTriageLoading(false);
    }
  };

  const runIssueRouting = async () => {
    try {
      setIssueLoading(true);
      setIssueResult(await apiClient.aiRouteIssue(issueTitle, issueBody));
      await loadAll();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Issue routing failed.');
    } finally {
      setIssueLoading(false);
    }
  };

  return (
    <AppShell
      eyebrow="Decision intelligence / trace"
      title="Laya decision layer"
      description="Inspect every typed judgment, its confidence, latency, and operational outcome. The model advises; deterministic Go thresholds decide."
      actions={
        <>
          <div className={`flex min-h-10 items-center gap-2 border px-3 font-mono text-[10px] uppercase tracking-[0.14em] ${status?.configured ? 'border-[var(--signal)]/25 bg-[var(--signal)]/6 text-[var(--signal)]' : 'border-[var(--amber)]/25 bg-[var(--amber)]/6 text-[var(--amber)]'}`}>
            <span className="signal-dot" />
            {status?.configured ? 'Laya gateway live' : 'Heuristic safe mode'}
          </div>
          <button className="btn-secondary" onClick={loadAll}><RefreshCw className="size-3.5" />Refresh</button>
        </>
      }
    >
      {error && (
        <div className="mb-5 flex items-center gap-3 border border-[var(--amber)]/25 bg-[var(--amber)]/6 px-4 py-3 text-sm text-amber-100">
          <AlertTriangle className="size-4 shrink-0" />{error}
        </div>
      )}

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4" aria-label="Decision summary">
        <Metric label="Decisions" value={decisions.length} detail="retained traces" icon={<BrainCircuit className="size-4" />} tone="var(--cyan)" />
        <Metric label="Live provider" value={layaCount} detail={`${decisions.length - layaCount} fallback`} icon={<Radio className="size-4" />} tone="var(--signal)" />
        <Metric label="Mean latency" value={`${meanLatency}ms`} detail="current sample" icon={<Gauge className="size-4" />} tone="var(--amber)" />
        <Metric label="Urgent" value={urgentCount} detail="notifications" icon={<Bell className="size-4" />} tone="var(--danger)" />
      </section>

      <div className="mt-6 grid gap-6 xl:grid-cols-[minmax(0,1.35fr)_minmax(22rem,.65fr)]">
        <section className="panel min-w-0">
          <div className="panel-header flex-wrap">
            <div className="flex items-center gap-2.5">
              <BrainCircuit className="size-4 text-[var(--cyan)]" />
              <h2 className="panel-title">Decision stream</h2>
            </div>
            <select className="field w-auto min-w-44 py-2 font-mono text-[10px] uppercase" value={moduleFilter} onChange={(event) => setModuleFilter(event.target.value)} aria-label="Filter by decision module">
              <option value="all">All modules</option>
              {modules.map((module) => <option key={module} value={module}>{moduleLabels[module] ?? module}</option>)}
            </select>
          </div>
          {visibleDecisions.length === 0 ? (
            <Empty icon={<BrainCircuit className="size-8" />} label="No decisions captured yet" detail="Run a failing pipeline or use a playground below." />
          ) : (
            <div className="max-h-[38rem] overflow-y-auto">
              {visibleDecisions.map((decision) => (
                <button
                  key={decision.id}
                  className={`data-row w-full grid-cols-[auto_minmax(0,1fr)_auto] text-left ${selected?.id === decision.id ? 'bg-[var(--signal)]/5' : ''}`}
                  onClick={() => setSelected(decision)}
                >
                  <span className={`grid size-9 place-items-center border ${decision.provider === 'laya' ? 'border-[var(--signal)]/20 bg-[var(--signal)]/7 text-[var(--signal)]' : 'border-[var(--amber)]/20 bg-[var(--amber)]/7 text-[var(--amber)]'}`}>
                    {decision.provider === 'laya' ? <Sparkles className="size-4" /> : <Gauge className="size-4" />}
                  </span>
                  <span className="min-w-0">
                    <span className="flex flex-wrap items-center gap-2">
                      <span className="truncate text-sm font-medium text-white">{moduleLabels[decision.context.module] ?? decision.context.module}</span>
                      <Provider provider={decision.provider} />
                    </span>
                    <span className="mt-1 block font-mono text-[10px] text-white/38">{decision.latency_ms}ms · {formatRelativeTime(new Date(decision.ts * 1000).toISOString())}</span>
                  </span>
                  <ArrowRight className="size-4 text-white/28" />
                </button>
              ))}
            </div>
          )}
        </section>

        <section className="panel min-w-0">
          <div className="panel-header">
            <h2 className="panel-title">Decision anatomy</h2>
            {selected && <Provider provider={selected.provider} />}
          </div>
          {selected ? <DecisionDetail decision={selected} /> : <Empty icon={<Route className="size-8" />} label="Select a decision" detail="Its inputs and typed answers will appear here." />}
        </section>
      </div>

      <section className="panel mt-6 min-w-0">
        <div className="panel-header">
          <div className="flex items-center gap-2.5">
            <Bell className="size-4 text-[var(--amber)]" />
            <h2 className="panel-title">Notification routing</h2>
          </div>
          <span className="font-mono text-[10px] text-white/35">{notifications.length} EVENTS</span>
        </div>
        {notifications.length === 0 ? (
          <Empty icon={<Bell className="size-8" />} label="No routed notifications" detail="Completion events will appear here." />
        ) : (
          <div className="grid md:grid-cols-2 xl:grid-cols-3">
            {notifications.slice(0, 9).map((notification) => (
              <article key={notification.id} className="border-b border-r border-white/8 p-4">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium text-white">{notification.pipeline_name}</p>
                    <p className="mt-1 text-xs leading-5 text-[var(--muted)]">{notification.message}</p>
                  </div>
                  <span className={levelStyle(notification.level)}>{notification.level}</span>
                </div>
                <p className="mt-3 font-mono text-[10px] text-white/35">urgency {(notification.urgent_probability * 100).toFixed(0)}%</p>
              </article>
            ))}
          </div>
        )}
      </section>

      <section className="mt-6 grid gap-6 xl:grid-cols-2" aria-label="Decision playgrounds">
        <Playground title="Log triage" icon={<FlaskConical className="size-4" />} hint="Severity · action · paging">
          <textarea className="field min-h-36 resize-y font-mono leading-6" value={triageInput} onChange={(event) => setTriageInput(event.target.value)} aria-label="Logs to triage" />
          <div className="mt-3 flex flex-wrap items-center justify-between gap-3">
            <button className="btn-primary" onClick={runTriage} disabled={triageLoading || !triageInput.trim()}>
              {triageLoading ? <LoaderCircle className="size-4 animate-spin" /> : <Sparkles className="size-4" />} Triage logs
            </button>
            {triageResult && <Result text={`${triageResult.action} · severity ${triageResult.severity?.toFixed(1) ?? '—'} · page ${(triageResult.page_probability * 100).toFixed(0)}%`} />}
          </div>
        </Playground>

        <Playground title="Issue router" icon={<Route className="size-4" />} hint="Component · labels · severity">
          <input className="field" value={issueTitle} onChange={(event) => setIssueTitle(event.target.value)} aria-label="Issue title" />
          <textarea className="field mt-2 min-h-24 resize-y" value={issueBody} onChange={(event) => setIssueBody(event.target.value)} aria-label="Issue body" />
          <div className="mt-3 flex flex-wrap items-center justify-between gap-3">
            <button className="btn-primary" onClick={runIssueRouting} disabled={issueLoading || !issueTitle.trim()}>
              {issueLoading ? <LoaderCircle className="size-4 animate-spin" /> : <Sparkles className="size-4" />} Route issue
            </button>
            {issueResult && <Result text={`${issueResult.component} · ${issueResult.labels.join(', ')}`} />}
          </div>
        </Playground>
      </section>
    </AppShell>
  );
}

function DecisionDetail({ decision }: { decision: Decision }) {
  return (
    <div className="max-h-[38rem] overflow-y-auto p-4">
      <div className="mb-5 grid grid-cols-2 gap-2">
        {Object.entries(decision.context).map(([key, value]) => (
          <div key={key} className="border border-white/8 bg-black/15 p-2.5">
            <p className="font-mono text-[9px] uppercase tracking-[0.13em] text-white/30">{key}</p>
            <p className="mt-1 truncate text-xs text-white/72">{String(value)}</p>
          </div>
        ))}
      </div>
      <p className="panel-title mb-2">Typed answers</p>
      <div className="space-y-2">
        {Object.entries(decision.questions).map(([name, question]) => {
          const answer = decision.answers[name] ?? {};
          const summary = answerSummary(answer);
          return (
            <article key={name} className="border border-white/8 bg-black/12 p-3">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="font-mono text-[10px] uppercase tracking-[0.12em] text-[var(--cyan)]">{name.replaceAll('_', ' ')}</p>
                  <p className="mt-1.5 text-xs leading-5 text-[var(--muted)]">{question.instructions}</p>
                </div>
                <span className="shrink-0 text-sm font-semibold text-white">{summary.label}</span>
              </div>
              {summary.confidence != null && (
                <div className="mt-3 h-1 overflow-hidden bg-white/8">
                  <div className="h-full bg-[var(--signal)]" style={{ width: `${Math.max(2, summary.confidence * 100)}%` }} />
                </div>
              )}
            </article>
          );
        })}
      </div>
      <details className="mt-4 border border-white/8 bg-black/15 p-3">
        <summary className="cursor-pointer font-mono text-[10px] uppercase tracking-[0.14em] text-white/45">Raw input state</summary>
        <pre className="mt-3 overflow-x-auto whitespace-pre-wrap font-mono text-[10px] leading-5 text-white/55">{JSON.stringify(decision.state, null, 2)}</pre>
      </details>
    </div>
  );
}

function answerSummary(answer: Record<string, unknown>): { label: string; confidence: number | null } {
  if (typeof answer.choice === 'string') {
    const probabilities = typeof answer.probabilities === 'object' && answer.probabilities ? answer.probabilities as Record<string, unknown> : {};
    const confidence = typeof probabilities[answer.choice] === 'number' ? probabilities[answer.choice] as number : null;
    return { label: answer.choice, confidence };
  }
  if (typeof answer.score === 'number') return { label: answer.score.toFixed(1), confidence: null };
  if (typeof answer.probability === 'number') return { label: `${Math.round(answer.probability * 100)}%`, confidence: answer.probability };
  return { label: '—', confidence: null };
}

function Provider({ provider }: { provider: Decision['provider'] }) {
  return <span className={`border px-2 py-0.5 font-mono text-[9px] uppercase tracking-[0.12em] ${provider === 'laya' ? 'border-[var(--signal)]/20 bg-[var(--signal)]/7 text-[var(--signal)]' : 'border-[var(--amber)]/20 bg-[var(--amber)]/7 text-[var(--amber)]'}`}>{provider}</span>;
}

function Metric({ label, value, detail, icon, tone }: { label: string; value: string | number; detail: string; icon: React.ReactNode; tone: string }) {
  return (
    <article className="metric fade-up" style={{ color: tone }}>
      <div className="flex items-center justify-between"><p className="font-mono text-[10px] uppercase tracking-[0.16em] text-white/42">{label}</p>{icon}</div>
      <p className="mt-4 text-3xl font-semibold tracking-[-0.04em] text-white">{value}</p>
      <p className="mt-1 font-mono text-[10px] uppercase tracking-[0.11em] text-white/36">{detail}</p>
    </article>
  );
}

function Playground({ title, hint, icon, children }: { title: string; hint: string; icon: React.ReactNode; children: React.ReactNode }) {
  return (
    <section className="panel">
      <div className="panel-header"><div className="flex items-center gap-2.5 text-[var(--signal)]">{icon}<h2 className="panel-title">{title}</h2></div><span className="font-mono text-[9px] text-white/30">{hint}</span></div>
      <div className="p-4">{children}</div>
    </section>
  );
}

function Result({ text }: { text: string }) {
  return <span className="flex items-center gap-2 font-mono text-[10px] uppercase tracking-[0.08em] text-[var(--signal)]"><CheckCircle2 className="size-3.5" />{text}</span>;
}

function Empty({ icon, label, detail }: { icon: React.ReactNode; label: string; detail: string }) {
  return <div className="grid min-h-64 place-items-center p-8 text-center"><div className="text-white/20">{icon}<p className="mt-4 text-sm font-medium text-white">{label}</p><p className="mt-1 text-xs text-[var(--muted)]">{detail}</p></div></div>;
}

function levelStyle(level: Notification['level']): string {
  if (level === 'urgent') return 'border border-[var(--danger)]/20 bg-[var(--danger)]/7 px-2 py-0.5 font-mono text-[9px] uppercase text-[var(--danger)]';
  if (level === 'notify') return 'border border-[var(--cyan)]/20 bg-[var(--cyan)]/7 px-2 py-0.5 font-mono text-[9px] uppercase text-[var(--cyan)]';
  return 'border border-white/10 bg-white/4 px-2 py-0.5 font-mono text-[9px] uppercase text-white/45';
}
