import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Header } from '../Header';
import { BackendHealth } from '@/types/logs';

// Mock ThemeToggle as it might depend on global context or matchMedia
vi.mock('@/components/ThemeToggle', () => ({
  ThemeToggle: () => <button data-testid="theme-toggle">ThemeToggle</button>
}));

describe('Header Component', () => {
  const mockOnSettingsClick = vi.fn();
  const mockOnReconnect = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders branding and connection status when disconnected', () => {
    render(
      <Header 
        health={null} 
        status="error" 
        error="Connection failed" 
        onSettingsClick={mockOnSettingsClick} 
        onReconnect={mockOnReconnect} 
      />
    );

    expect(screen.getByText(/LOG/i)).toBeInTheDocument();
    expect(screen.getByText('Connection failed')).toBeInTheDocument();
  });

  it('renders health indicators when connected', () => {
    const mockHealth: BackendHealth = {
      status: 'healthy',
      ingestionRate: 500,
      chunksCount: 1200,
      storageUsed: 2048000, // approx 2MB
      uptime: 3600
    };

    render(
      <Header 
        health={mockHealth} 
        status="connected" 
        error={null} 
        onSettingsClick={mockOnSettingsClick} 
        onReconnect={mockOnReconnect} 
      />
    );

    expect(screen.getByText('Connected to backend')).toBeInTheDocument();
    expect(screen.getByText('500/s')).toBeInTheDocument();
    expect(screen.getByText('1200')).toBeInTheDocument();
    expect(screen.getByText('2 MB')).toBeInTheDocument();
  });

  it('triggers onSettingsClick when settings button is clicked', async () => {
    const user = userEvent.setup();
    render(
      <Header 
        health={null} 
        status="error" 
        error={null} 
        onSettingsClick={mockOnSettingsClick} 
        onReconnect={mockOnReconnect} 
      />
    );

    const settingsBtn = screen.getByRole('button', { name: /settings/i });
    await user.click(settingsBtn);

    expect(mockOnSettingsClick).toHaveBeenCalledTimes(1);
  });
});
