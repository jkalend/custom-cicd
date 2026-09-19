'use client';

import { useCallback, useEffect, useState } from 'react';
import {
  BeakerIcon,
  BellIcon,
  BoltIcon,
  ClipboardDocumentCheckIcon,
  ArrowPathIcon,
} from '@heroicons/react/24/outline';

import { apiClient } from '@/lib/api';
import { formatRelativeTime, cn } from '@/lib/utils';
import { LogViewer, useLogs } from '@/components/ui/log-viewer';
import type {
  AIStatus,
  Decision,
  IssueRouteResult,
  Notification,
  TriageResult,
} from '@/types/api';

const moduleLabels: Record<string, string> = {
  classify_failure: 'Failure classification',
  triage_logs: 'Log triage',
  route_issue: 'Issue routing',
  filter_notification: 'Notification filter',
};

function providerBadge(provider: Decision['provider']): string {
  if (provider === 'jev') return 'bg-purple-100 text-purple-800 border-purple-200';
  return 'bg-gray-100 text-gray-600 border-gray-200';
}

function levelColor(level: Notification['level']): string {
  switch (level) {
    case 'urgent':
      return 'bg-red-100 text-red-800 border-red-200';
    case 'notify':
      return 'bg-blue-100 text-blue-800 border-blue-200';
    case 'defer':
      return 'bg-yellow-100 text-yellow-800 border-yellow-200';
    default:
      return 'bg-gray-100 text-gray-600 border-gray-200';
  }
}

export default function AIDashboard() {
  const [status, setStatus] = useState<AIStatus | null>(null);
  const [decisions, setDecisions] = useState<Decision[]>([]);
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [selected, setSelected] = useState<Decision | null>(null);

  // Log triage playground
  const [triageInput, setTriageInput] = useState(
    'ERROR 2026-09-20 connection to database reset after 3 retries\nOutOfMemoryError: Java heap space in worker-2'
  );
  const [triageResult, setTriageResult] = useState<TriageResult | null>(null);
  const [triageLoading, setTriageLoading] = useState(false);

  // Issue routing playground
  const [issueTitle, setIssueTitle] = useState(
    'After updating to 2.4, Windows crashes when opening settings'
  );
  const [issueBody, setIssueBody] = useState('Crash happens only on first launch after the update.');
  const [issueResult, setIssueResult] = useState<IssueRouteResult | null>(null);
  const [issueLoading, setIssueLoading] = useState(false);

  const { logs, addLog, clearLogs } = useLogs();

  const loadAll = useCallback(async () => {
    try {
      const [statusData, decisionsData, notificationsData] = await Promise.all([
        apiClient.aiStatus(),
        apiClient.aiDecisions(30),
        apiClient.aiNotifications(),
      ]);
      setStatus(statusData);
      setDecisions(Array.isArray(decisionsData) ? decisionsData : []);
      setNotifications(Array.isArray(notificationsData) ? notificationsData : []);
    } catch (err) {
      addLog(`❌ Failed to load AI data: ${err instanceof Error ? err.message : 'unknown'}`, 'error');
    }
  }, [addLog]);

  useEffect(() => {
    loadAll();
    const interval = setInterval(loadAll, 5000);
    return () => clearInterval(interval);
  }, [loadAll]);

  const runTriage = async () => {
    try {
      setTriageLoading(true);
      const result = await apiClient.aiTriage(triageInput, 'playground');
      setTriageResult(result);
      addLog(`🔬 Log triage: ${result.action} (page p=${result.page_probability.toFixed(2)})`, 'info');
    } catch (err) {
      addLog(`❌ Triage failed: ${err instanceof Error ? err.message : 'unknown'}`, 'error');
    } finally {
      setTriageLoading(false);
    }
  };

  const runIssueRouting = async () => {
    try {
      setIssueLoading(true);
      const result = await apiClient.aiRouteIssue(issueTitle, issueBody);
      setIssueResult(result);
      addLog(`🏷️ Issue routed: ${result.component} → [${result.labels.join(', ')}]`, 'info');
    } catch (err) {
      addLog(`❌ Issue routing failed: ${err instanceof Error ? err.message : 'unknown'}`, 'error');
    } finally {
      setIssueLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header */}
        <div className="bg-gray-900 text-white rounded-lg p-6 mb-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <BoltIcon className="w-8 h-8" />
              <div>
                <h1 className="text-3xl font-bold">AI Decision Layer</h1>
                <p className="text-gray-300">Jev-powered judgments across your CI/CD</p>
              </div>
            </div>
            <div className="text-right text-sm">
              {status && (
                <div
                  className={cn(
                    'inline-flex items-center gap-2 px-3 py-1.5 rounded-full border font-medium',
                    status.configured
                      ? 'bg-green-900/40 text-green-300 border-green-700'
                      : 'bg-gray-800 text-gray-400 border-gray-600'
                  )}
                >
                  {status.configured ? '⚡ Jev live' : '🔄 heuristic mode (no key)'}
                </div>
              )}
              <p className="text-gray-400 mt-1">{status?.model ?? '…'}</p>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6">
          {/* Decision log */}
          <div className="lg:col-span-2 bg-white rounded-lg shadow p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-semibold text-gray-900">
                🧠 Decisions ({decisions.length})
              </h2>
              <button
                onClick={loadAll}
                className="flex items-center gap-2 text-blue-600 hover:text-blue-700"
              >
                <ArrowPathIcon className="w-4 h-4" />
                Refresh
              </button>
            </div>

            {decisions.length === 0 ? (
              <p className="text-gray-700">
                No decisions yet — run a pipeline with a failing step, or use the playgrounds below.
              </p>
            ) : (
              <div className="space-y-2 max-h-[420px] overflow-y-auto">
                {decisions.map((d) => (
                  <button
                    key={d.id}
                    onClick={() => setSelected(d)}
                    className={cn(
                      'w-full text-left border rounded p-3 hover:border-blue-400 transition-colors',
                      selected?.id === d.id ? 'border-blue-500 bg-blue-50' : 'border-gray-200 bg-gray-50'
                    )}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-gray-900">
                          {moduleLabels[d.context.module] ?? d.context.module}
                        </span>
                        <span
                          className={cn(
                            'text-xs px-2 py-0.5 rounded-full border font-medium',
                            providerBadge(d.provider)
                          )}
                        >
                          {d.provider}
                        </span>
                      </div>
                      <span className="text-xs text-gray-800">
                        {d.latency_ms}ms · {formatRelativeTime(new Date(d.ts * 1000).toISOString())}
                      </span>
                    </div>
                    {d.error && <p className="text-xs text-red-600 mt-1">⚠ {d.error}</p>}
                  </button>
                ))}
              </div>
            )}

            {/* Selected decision detail */}
            {selected && (
              <div className="mt-4 border border-gray-200 rounded p-4 bg-gray-900 text-gray-100 font-mono text-xs overflow-x-auto">
                <p className="text-gray-400 mb-2">state + questions → answers</p>
                <pre className="whitespace-pre-wrap">
                  {JSON.stringify(
                    { state: selected.state, questions: selected.questions, answers: selected.answers },
                    null,
                    2
                  )}
                </pre>
              </div>
            )}
          </div>

          {/* Notifications */}
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center gap-2 mb-4">
              <BellIcon className="w-5 h-5 text-gray-700" />
              <h2 className="text-xl font-semibold text-gray-900">Notifications</h2>
            </div>
            {notifications.length === 0 ? (
              <p className="text-gray-700">No notifications yet.</p>
            ) : (
              <div className="space-y-2 max-h-[420px] overflow-y-auto">
                {notifications.map((n) => (
                  <div key={n.id} className="border border-gray-200 rounded p-3 bg-gray-50">
                    <div className="flex items-center justify-between mb-1">
                      <span className="font-medium text-gray-900 text-sm">{n.pipeline_name}</span>
                      <span
                        className={cn(
                          'text-xs px-2 py-0.5 rounded-full border font-medium',
                          levelColor(n.level)
                        )}
                      >
                        {n.level}
                      </span>
                    </div>
                    <p className="text-xs text-gray-800">{n.title}</p>
                    <p className="text-xs text-gray-600 mt-1">{n.message}</p>
                    <p className="text-xs text-gray-500 mt-1">
                      urgency p={n.urgent_probability.toFixed(2)}
                    </p>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Playgrounds */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
          {/* Log triage */}
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center gap-2 mb-4">
              <BeakerIcon className="w-5 h-5 text-gray-700" />
              <h2 className="text-xl font-semibold text-gray-900">Log Triage</h2>
            </div>
            <textarea
              value={triageInput}
              onChange={(e) => setTriageInput(e.target.value)}
              className="w-full h-32 p-3 border border-gray-300 rounded-lg font-mono text-sm text-gray-900"
              placeholder="Paste log output..."
            />
            <button
              onClick={runTriage}
              disabled={triageLoading}
              className="mt-3 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white px-4 py-2 rounded-lg"
            >
              {triageLoading ? 'Thinking…' : 'Triage logs'}
            </button>
            {triageResult && (
              <div className="mt-4 border border-gray-200 rounded p-4 bg-gray-50">
                <p className="text-sm text-gray-900">
                  <strong>Action:</strong> {triageResult.action} · page p={triageResult.page_probability.toFixed(2)}
                </p>
                <p className="text-sm text-gray-900">
                  <strong>Severity:</strong> {triageResult.severity ?? '—'}
                </p>
              </div>
            )}
          </div>

          {/* Issue routing */}
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center gap-2 mb-4">
              <ClipboardDocumentCheckIcon className="w-5 h-5 text-gray-700" />
              <h2 className="text-xl font-semibold text-gray-900">Issue Router</h2>
            </div>
            <input
              value={issueTitle}
              onChange={(e) => setIssueTitle(e.target.value)}
              className="w-full p-3 border border-gray-300 rounded-lg text-sm text-gray-900"
              placeholder="Issue title"
            />
            <textarea
              value={issueBody}
              onChange={(e) => setIssueBody(e.target.value)}
              className="w-full h-20 p-3 mt-2 border border-gray-300 rounded-lg text-sm text-gray-900"
              placeholder="Issue body (optional)"
            />
            <button
              onClick={runIssueRouting}
              disabled={issueLoading}
              className="mt-3 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white px-4 py-2 rounded-lg"
            >
              {issueLoading ? 'Thinking…' : 'Route issue'}
            </button>
            {issueResult && (
              <div className="mt-4 border border-gray-200 rounded p-4 bg-gray-50">
                <p className="text-sm text-gray-900">
                  <strong>Component:</strong> {issueResult.component} · is_bug p={issueResult.is_bug.toFixed(2)}
                </p>
                <div className="flex flex-wrap gap-1 mt-2">
                  {issueResult.labels.map((l) => (
                    <span
                      key={l}
                      className="text-xs bg-blue-100 text-blue-800 border border-blue-200 px-2 py-0.5 rounded-full"
                    >
                      {l}
                    </span>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>

        <LogViewer logs={logs} onClear={clearLogs} height="h-40" />
      </div>
    </div>
  );
}
