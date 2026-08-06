import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { LogLevelBadge } from '../LogLevelBadge';

describe('LogLevelBadge Component', () => {
  it('renders the level text', () => {
    render(<LogLevelBadge level="error" />);
    expect(screen.getByText('error')).toBeInTheDocument();
  });

  it('applies the uppercase class so the text is styled as uppercase', () => {
    render(<LogLevelBadge level="error" />);
    const badge = screen.getByText('error');
    expect(badge.className).toContain('uppercase');
  });

  it('applies destructive (red) styling for the error level', () => {
    render(<LogLevelBadge level="error" />);
    const badge = screen.getByText('error');
    expect(badge.className).toContain('text-destructive');
  });

  it('applies warning (orange) styling for the warn level', () => {
    render(<LogLevelBadge level="warn" />);
    const badge = screen.getByText('warn');
    expect(badge.className).toContain('text-warning');
  });

  it('applies info (blue) styling for the info level', () => {
    render(<LogLevelBadge level="info" />);
    const badge = screen.getByText('info');
    expect(badge.className).toContain('text-info');
  });

  it('applies muted styling for the debug level', () => {
    render(<LogLevelBadge level="debug" />);
    const badge = screen.getByText('debug');
    expect(badge.className).toContain('text-muted-foreground');
  });

  it('merges any extra className passed in', () => {
    render(<LogLevelBadge level="info" className="w-[60px]" />);
    const badge = screen.getByText('info');
    expect(badge.className).toContain('w-[60px]');
  });
});
