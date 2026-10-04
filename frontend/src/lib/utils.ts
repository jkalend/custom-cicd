// Shared UI helpers. Like api.ts, this file was imported by every page but
// never committed in the original repo — restored here.

export function cn(...classes: Array<string | false | null | undefined>): string {
  return classes.filter(Boolean).join(' ');
}

export function formatDate(dateStr: string): string {
  if (!dateStr) return '—';
  const date = new Date(dateStr);
  if (Number.isNaN(date.getTime())) return dateStr;
  return date.toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function formatRelativeTime(dateStr: string): string {
  if (!dateStr) return '—';
  const date = new Date(dateStr);
  if (Number.isNaN(date.getTime())) return dateStr;

  const seconds = Math.max(0, Math.floor((Date.now() - date.getTime()) / 1000));
  if (seconds < 5) return 'just now';
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}d ago`;
  return formatDate(dateStr);
}

export function formatDuration(seconds?: number | null): string {
  if (seconds == null) return '—';
  if (seconds < 1) return `${(seconds * 1000).toFixed(0)}ms`;
  if (seconds < 60) return `${seconds.toFixed(1)}s`;
  const minutes = Math.floor(seconds / 60);
  const rest = Math.round(seconds % 60);
  return `${minutes}m ${rest}s`;
}

export async function copyToClipboard(text: string): Promise<void> {
  await navigator.clipboard.writeText(text);
}

export function getStatusEmoji(status: string): string {
  switch (status) {
    case 'success':
    case 'healthy':
      return '✅';
    case 'failed':
    case 'error':
      return '❌';
    case 'running':
      return '🏃';
    case 'pending':
      return '⏳';
    case 'cancelled':
      return '🛑';
    case 'skipped':
      return '⏭️';
    case 'never_run':
      return '💤';
    default:
      return '❓';
  }
}

export function getStatusColor(status: string): string {
  switch (status) {
    case 'success':
    case 'healthy':
      return 'bg-[rgba(184,255,92,0.08)] text-[var(--signal)] border-[rgba(184,255,92,0.22)]';
    case 'failed':
    case 'error':
      return 'bg-[rgba(255,102,115,0.08)] text-[var(--danger)] border-[rgba(255,102,115,0.22)]';
    case 'running':
      return 'bg-[rgba(97,217,255,0.08)] text-[var(--cyan)] border-[rgba(97,217,255,0.22)]';
    case 'pending':
      return 'bg-[rgba(255,191,105,0.08)] text-[var(--amber)] border-[rgba(255,191,105,0.22)]';
    case 'cancelled':
      return 'bg-white/4 text-white/55 border-white/12';
    case 'skipped':
      return 'bg-white/3 text-white/45 border-white/10';
    case 'never_run':
      return 'bg-white/3 text-white/40 border-white/10';
    default:
      return 'bg-white/4 text-white/55 border-white/12';
  }
}

export const defaultPipelineTemplate = {
  name: 'My Pipeline',
  version: '1.0.0',
  description: 'A pipeline with a couple of steps',
  variables: {
    PROJECT_NAME: 'my-app',
  },
  steps: [
    {
      name: 'Say Hello',
      command: 'echo "Hello from ${PROJECT_NAME}!"',
      timeout: 30,
      retry_count: 0,
      continue_on_error: false,
    },
    {
      name: 'Run Tests',
      command: 'echo "running tests..."',
      timeout: 120,
      retry_count: 1,
      continue_on_error: false,
    },
  ],
};
