import { LogEntry } from '@/types/logs';
import {
  Download,
  FileJson,
  FileSpreadsheet,
  Copy,
} from 'lucide-react';

import { Button } from './ui/button';

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from './ui/dropdown-menu';

import { toast } from 'sonner';

interface LogExportProps {
  logs: LogEntry[];
  disabled?: boolean;
}

export function LogExport({
  logs,
  disabled,
}: LogExportProps) {

  const exportAsJSON = () => {
    if (logs.length === 0) {
      toast.error('No logs to export');
      return;
    }

    const data = JSON.stringify(logs, null, 2);

    downloadFile(
      data,
      'logs.json',
      'application/json'
    );

    toast.success(
      `Exported ${logs.length} logs as JSON`
    );
  };

  const exportAsCSV = () => {
    if (logs.length === 0) {
      toast.error('No logs to export');
      return;
    }

    // Collect all unique label keys
    const allLabelKeys = new Set<string>();

    logs.forEach((log) => {
      Object.keys(log.labels ?? {}).forEach((key) =>
        allLabelKeys.add(key)
      );
    });

    const labelKeys = Array.from(allLabelKeys).sort();

    // CSV Headers
    const headers = [
      'id',
      'timestamp',
      'level',
      'message',
      ...labelKeys.map(
        (key) => `label_${key}`
      ),
    ];

    // CSV Rows
    const rows = logs.map((log) => {
      const baseFields = [
        escapeCSV(log.id),
        escapeCSV(log.timestamp),
        escapeCSV(log.level),
        escapeCSV(log.message),
      ];

      const labelFields = labelKeys.map((key) =>
        escapeCSV(log.labels?.[key] ?? '')
      );

      return [
        ...baseFields,
        ...labelFields,
      ].join(',');
    });

    const csvContent = [
      headers.join(','),
      ...rows,
    ].join('\n');

    downloadFile(
      csvContent,
      'logs.csv',
      'text/csv;charset=utf-8;'
    );

    toast.success(
      `Exported ${logs.length} logs as CSV`
    );
  };

  const copyAllLogs = async () => {
    if (logs.length === 0) {
      toast.error('No logs to copy');
      return;
    }

    try {
      const formattedLogs = logs
        .map(
          (log) =>
            `[${log.timestamp}] ${log.level.toUpperCase()} - ${log.message}`
        )
        .join('\n');

      await navigator.clipboard.writeText(
        formattedLogs
      );

      toast.success(
        `Copied ${logs.length} logs to clipboard`
      );
    } catch (error) {
      toast.error('Failed to copy logs');
    }
  };

  const escapeCSV = (
    value: unknown
  ): string => {
    if (
      value === null ||
      value === undefined
    ) {
      return '';
    }

    const str = String(value);

    if (
      str.includes(',') ||
      str.includes('"') ||
      str.includes('\n')
    ) {
      return `"${str.replace(/"/g, '""')}"`;
    }

    return str;
  };

  const downloadFile = (
    content: string,
    filename: string,
    mimeType: string
  ) => {
    const blob = new Blob([content], {
      type: mimeType,
    });

    const url = URL.createObjectURL(blob);

    const link =
      document.createElement('a');

    link.href = url;
    link.download = filename;

    document.body.appendChild(link);

    link.click();

    document.body.removeChild(link);

    URL.revokeObjectURL(url);
  };

  return (
    <div className="flex items-center gap-2">

      {/* Export Dropdown */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="outline"
            size="sm"
            disabled={
              disabled || logs.length === 0
            }
            className="gap-2"
          >
            <Download className="h-4 w-4" />
            Export
          </Button>
        </DropdownMenuTrigger>

        <DropdownMenuContent align="end">

          <DropdownMenuItem
            onClick={exportAsJSON}
            className="gap-2 cursor-pointer"
          >
            <FileJson className="h-4 w-4" />
            Export as JSON
          </DropdownMenuItem>

          <DropdownMenuItem
            onClick={exportAsCSV}
            className="gap-2 cursor-pointer"
          >
            <FileSpreadsheet className="h-4 w-4" />
            Export as CSV
          </DropdownMenuItem>

        </DropdownMenuContent>
      </DropdownMenu>

      
      <Button
        variant="outline"
        size="sm"
        onClick={copyAllLogs}
        disabled={
          disabled || logs.length === 0
        }
        className="gap-2"
      >
        <Copy className="h-4 w-4" />
        Copy Logs
      </Button>

    </div>
  );
}