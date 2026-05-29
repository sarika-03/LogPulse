import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryBar } from '../QueryBar';

describe('QueryBar Component', () => {
  const mockOnQuery = vi.fn();
  const mockOnRefresh = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders query input and buttons correctly', () => {
    render(
      <QueryBar 
        onQuery={mockOnQuery} 
        onRefresh={mockOnRefresh} 
        isLoading={false} 
        isConnected={true} 
      />
    );

    expect(screen.getByPlaceholderText('{service="api-gateway", env="prod"}')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /run query/i })).toBeInTheDocument();
    expect(screen.getByText('1h')).toBeInTheDocument();
  });

  it('handles input changes correctly', async () => {
    const user = userEvent.setup();
    render(
      <QueryBar 
        onQuery={mockOnQuery} 
        onRefresh={mockOnRefresh} 
        isLoading={false} 
        isConnected={true} 
      />
    );

    const input = screen.getByPlaceholderText('{service="api-gateway", env="prod"}');
    fireEvent.change(input, { target: { value: '{level="error"}' } });
    
    expect(input).toHaveValue('{level="error"}');
  });

  it('submits query with correct parameters on button click', async () => {
    const user = userEvent.setup();
    render(
      <QueryBar 
        onQuery={mockOnQuery} 
        onRefresh={mockOnRefresh} 
        isLoading={false} 
        isConnected={true} 
      />
    );

    const input = screen.getByPlaceholderText('{service="api-gateway", env="prod"}');
    fireEvent.change(input, { target: { value: '{level="error"}' } });
    
    const runBtn = screen.getByRole('button', { name: /run query/i });
    await user.click(runBtn);

    expect(mockOnQuery).toHaveBeenCalledWith('{level="error"}', '1h');
    expect(mockOnQuery).toHaveBeenCalledTimes(1);
  });

  it('disables interactions when disconnected', async () => {
    const user = userEvent.setup();
    render(
      <QueryBar 
        onQuery={mockOnQuery} 
        onRefresh={mockOnRefresh} 
        isLoading={false} 
        isConnected={false} 
      />
    );

    const input = screen.getByPlaceholderText('{service="api-gateway", env="prod"}');
    expect(input).toBeDisabled();

    const runBtn = screen.getByRole('button', { name: /run query/i });
    expect(runBtn).toBeDisabled();
    
    await user.click(runBtn);
    expect(mockOnQuery).not.toHaveBeenCalled();
  });

  it('shows loading state and prevents submission when loading', async () => {
    const user = userEvent.setup();
    render(
      <QueryBar 
        onQuery={mockOnQuery} 
        onRefresh={mockOnRefresh} 
        isLoading={true} 
        isConnected={true} 
      />
    );

    const runBtn = screen.getByRole('button', { name: /run query/i });
    expect(runBtn).toBeDisabled();
    
    await user.click(runBtn);
    expect(mockOnQuery).not.toHaveBeenCalled();
  });

  it('calls onRefresh when refresh button is clicked', async () => {
    const user = userEvent.setup();
    render(
      <QueryBar 
        onQuery={mockOnQuery} 
        onRefresh={mockOnRefresh} 
        isLoading={false} 
        isConnected={true} 
      />
    );

    const refreshBtn = screen.getByRole('button', { name: /refresh/i });
    await user.click(refreshBtn);

    expect(mockOnRefresh).toHaveBeenCalledTimes(1);
  });

  it('updates selected time range', async () => {
    const user = userEvent.setup();
    render(
      <QueryBar 
        onQuery={mockOnQuery} 
        onRefresh={mockOnRefresh} 
        isLoading={false} 
        isConnected={true} 
      />
    );

    const rangeBtn = screen.getByText('15m');
    await user.click(rangeBtn);
    
    const runBtn = screen.getByRole('button', { name: /run query/i });
    await user.click(runBtn);

    // Default query is '{service="api-gateway"}'
    expect(mockOnQuery).toHaveBeenCalledWith('{service="api-gateway"}', '15m');
  });
});
