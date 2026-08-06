import { LogLevel } from '@/types/logs';
import { cn } from '@/lib/utils';

/**
 * Tailwind classes for each log level.
 *
 * Each level gets a soft background tint, a matching text color, and a
 * matching border color, all driven by the theme's CSS variables
 * (see `--destructive`, `--warning`, `--info` and `--muted` in
 * `src/index.css`) so the badge automatically follows the cyberpunk
 * dark theme.
 */
const LOG_LEVEL_STYLES: Record<LogLevel, string> = {
  error: 'bg-destructive/20 text-destructive border-destructive/30',
  warn: 'bg-warning/20 text-warning border-warning/30',
  info: 'bg-info/20 text-info border-info/30',
  debug: 'bg-muted text-muted-foreground border-border',
};

/**
 * Props for the {@link LogLevelBadge} component.
 */
export interface LogLevelBadgeProps {
  /** The log level to display (error, warn, info, or debug). */
  level: LogLevel;
  /** Optional extra Tailwind classes to merge onto the badge. */
  className?: string;
}

/**
 * LogLevelBadge
 *
 * Renders a small, color-coded, uppercase badge for a log's severity
 * level (ERROR / WARN / INFO / DEBUG). Colors follow the app's cyberpunk
 * theme CSS variables so they stay consistent everywhere logs are shown.
 *
 * This was extracted from the inline badge markup that used to live only
 * inside `LogViewer.tsx`, so it can now be reused by any component that
 * needs to show a log's level (e.g. `LiveStream`, analytics tooltips).
 *
 * @example
 * ```tsx
 * <LogLevelBadge level="error" />
 * ```
 */
export function LogLevelBadge({ level, className }: LogLevelBadgeProps) {
  return (
    <span
      className={cn(
        'text-xs font-mono uppercase px-2 py-0.5 rounded border inline-block text-center',
        LOG_LEVEL_STYLES[level],
        className,
      )}
    >
      {level}
    </span>
  );
}
