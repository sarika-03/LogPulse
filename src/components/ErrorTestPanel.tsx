import { useState } from 'react';
import { AlertTriangle, Bug, Send, Copy } from 'lucide-react';
import { toast } from 'sonner';

interface ErrorTest {
  name: string;
  description: string;
  query: string;
  expectedError: string;
  expectedCode: string;
}

const errorTests: ErrorTest[] = [
  {
    name: 'Invalid Query Syntax',
    description: 'Original issue - incomplete label assignment',
    query: '{service="api-gateway", level=}',
    expectedError: 'Invalid query syntax',
    expectedCode: 'BAD_QUERY'
  },
  {
    name: 'Empty Query',
    description: 'Missing query parameter',
    query: '',
    expectedError: 'Query parameter is required',
    expectedCode: 'VALIDATION_ERROR'
  },
  {
    name: 'Invalid Regex',
    description: 'Malformed regex pattern',
    query: '{service=~"[invalid"}',
    expectedError: 'Invalid regex pattern',
    expectedCode: 'INVALID_REGEX'
  },
  {
    name: 'Unmatched Braces',
    description: 'Missing closing brace',
    query: '{service="api"',
    expectedError: 'Invalid query syntax',
    expectedCode: 'BAD_QUERY'
  }
];

export function ErrorTestPanel() {
  const [selectedTest, setSelectedTest] = useState<ErrorTest | null>(null);
  const [testResult, setTestResult] = useState<any>(null);
  const [isLoading, setIsLoading] = useState(false);

  const runTest = async (test: ErrorTest) => {
    setIsLoading(true);
    setSelectedTest(test);
    setTestResult(null);

    try {
      const baseUrl = 'http://localhost:8080';
      const url = test.query 
        ? `${baseUrl}/query?query=${encodeURIComponent(test.query)}`
        : `${baseUrl}/query`;

      const response = await fetch(url);
      const result = await response.json();
      
      setTestResult({
        status: response.status,
        data: result,
        success: !response.ok && result.code === test.expectedCode
      });

      if (!response.ok && result.code === test.expectedCode) {
        toast.success(`✅ Test passed: ${test.name}`);
      } else {
        toast.error(`❌ Test failed: ${test.name}`);
      }
    } catch (error) {
      setTestResult({
        status: 'ERROR',
        data: { error: 'Network error - is backend running?' },
        success: false
      });
      toast.error('Network error - check if backend is running');
    } finally {
      setIsLoading(false);
    }
  };

  const copyResult = () => {
    if (testResult) {
      navigator.clipboard.writeText(JSON.stringify(testResult.data, null, 2));
      toast.success('Result copied to clipboard');
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2 mb-4">
        <Bug className="h-4 w-4 text-primary" />
        <h3 className="text-sm font-semibold">Error Handling Tests</h3>
      </div>

      <div className="space-y-2">
        {errorTests.map((test, index) => (
          <div key={index} className="border border-border rounded-lg p-3 space-y-2">
            <div className="flex items-start justify-between">
              <div className="flex-1">
                <h4 className="text-sm font-medium">{test.name}</h4>
                <p className="text-xs text-muted-foreground">{test.description}</p>
              </div>
              <button
                onClick={() => runTest(test)}
                disabled={isLoading}
                className="flex items-center gap-1 px-2 py-1 text-xs bg-primary text-primary-foreground rounded hover:opacity-90 disabled:opacity-50"
              >
                <Send className="h-3 w-3" />
                Test
              </button>
            </div>
            
            <div className="bg-muted/50 rounded p-2">
              <code className="text-xs font-mono">
                {test.query || '(empty query)'}
              </code>
            </div>

            <div className="text-xs text-muted-foreground">
              Expected: <span className="font-mono">{test.expectedCode}</span>
            </div>
          </div>
        ))}
      </div>

      {selectedTest && testResult && (
        <div className="border border-border rounded-lg p-3 space-y-3">
          <div className="flex items-center justify-between">
            <h4 className="text-sm font-semibold flex items-center gap-2">
              {testResult.success ? (
                <span className="text-green-500">✅</span>
              ) : (
                <AlertTriangle className="h-4 w-4 text-destructive" />
              )}
              Test Result: {selectedTest.name}
            </h4>
            <button
              onClick={copyResult}
              className="p-1 text-muted-foreground hover:text-foreground"
            >
              <Copy className="h-3 w-3" />
            </button>
          </div>

          <div className="space-y-2">
            <div className="text-xs">
              <span className="font-medium">Status:</span> {testResult.status}
            </div>
            
            <div className="bg-muted/50 rounded p-2">
              <pre className="text-xs font-mono overflow-auto">
                {JSON.stringify(testResult.data, null, 2)}
              </pre>
            </div>

            {testResult.success ? (
              <div className="text-xs text-green-600 bg-green-50 dark:bg-green-950 p-2 rounded">
                ✅ Test passed! Backend returned expected error structure.
              </div>
            ) : (
              <div className="text-xs text-destructive bg-destructive/10 p-2 rounded">
                ❌ Test failed. Check if backend is running with updated error handling.
              </div>
            )}
          </div>
        </div>
      )}

      <div className="text-xs text-muted-foreground bg-muted/30 p-3 rounded-lg">
        <div className="flex items-start gap-2">
          <AlertTriangle className="h-3 w-3 mt-0.5 text-warning" />
          <div>
            <p className="font-medium mb-1">Testing Instructions:</p>
            <ol className="space-y-1 text-xs">
              <li>1. Make sure the backend server is running on localhost:8080</li>
              <li>2. Click "Test" buttons to verify error handling improvements</li>
              <li>3. Green checkmarks indicate structured errors are working</li>
              <li>4. Copy results to include in your PR as proof</li>
            </ol>
          </div>
        </div>
      </div>
    </div>
  );
}