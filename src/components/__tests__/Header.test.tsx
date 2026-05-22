import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Header } from '../Header';

// Mock ThemeToggle as it might depend on global context or matchMedia
jest.mock('@/components/ThemeToggle', () => ({
  ThemeToggle: () => <button data-testid="theme-toggle">ThemeToggle</button>
}));

describe('Header Component', () => {
  const mockOnSettingsClick = jest.fn();
  const mockOnReconnect = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
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
    const mockHealth = {
      ingestionRate: 500,
      chunksCount: 1200,
      storageUsed: 2048000 // approx 2MB
    };

    render(
      <Header 
        health={mockHealth as any} 
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

    const buttons = screen.getAllByRole('button');
    // Last button is settings
    await user.click(buttons[buttons.length - 1]);

    expect(mockOnSettingsClick).toHaveBeenCalledTimes(1);
  });
});
