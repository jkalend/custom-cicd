import { PipelineStatus } from '@/types/api';
import { getStatusEmoji, getStatusColor, cn } from '@/lib/utils';

interface StatusBadgeProps {
  status: PipelineStatus;
  className?: string;
  showEmoji?: boolean;
}

export function StatusBadge({ 
  status, 
  className,
  showEmoji = true 
}: StatusBadgeProps) {
  const emoji = getStatusEmoji(status);
  const colorClasses = getStatusColor(status);

  return (
    <span 
      className={cn(
        'inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium border',
        colorClasses,
        className
      )}
    >
      {showEmoji && <span>{emoji}</span>}
      <span className="capitalize">{status.replace('_', ' ')}</span>
    </span>
  );
} 
