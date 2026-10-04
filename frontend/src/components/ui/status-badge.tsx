import { PipelineStatus } from '@/types/api';
import { getStatusColor, cn } from '@/lib/utils';

interface StatusBadgeProps {
  status: PipelineStatus;
  className?: string;
  showEmoji?: boolean;
}

export function StatusBadge({ 
  status, 
  className,
  showEmoji = false
}: StatusBadgeProps) {
  const safeStatus = status || 'never_run';
  const colorClasses = getStatusColor(safeStatus);

  return (
    <span 
      className={cn(
        'inline-flex items-center gap-2 border px-2.5 py-1 font-mono text-[10px] font-semibold uppercase tracking-[0.12em]',
        colorClasses,
        className
      )}
    >
      {showEmoji && <span aria-hidden="true">●</span>}
      <span className="signal-dot" aria-hidden="true" />
      <span>{safeStatus.replaceAll('_', ' ')}</span>
    </span>
  );
} 
