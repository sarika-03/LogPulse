import { useLocation } from "react-router-dom";
import { useEffect } from "react";
import { AlertTriangle, ArrowLeft, Home } from "lucide-react";

const NotFound = () => {
  const location = useLocation();

  useEffect(() => {
    console.error("404 Error:", location.pathname);
  }, [location.pathname]);

  return (
    <div className="min-h-screen bg-gradient-to-br from-background via-muted to-background flex items-center justify-center px-6">
      <div className="text-center max-w-md">

        {/* Icon */}
        <div className="flex justify-center mb-6">
          <div className="rounded-full bg-destructive/10 p-4">
            <AlertTriangle className="h-10 w-10 text-destructive" />
          </div>
        </div>

        {/* Error Code */}
        <h1 className="text-6xl font-bold tracking-tight text-foreground mb-2">
          404
        </h1>

        {/* Message */}
        <p className="text-xl text-muted-foreground mb-2">
          Page not found
        </p>

        <p className="text-sm text-muted-foreground mb-8">
          The route <span className="text-primary font-mono">{location.pathname}</span> doesn’t exist in this system.
        </p>

        {/* Buttons */}
        <div className="flex flex-col sm:flex-row gap-3 justify-center">
          <a
            href="/"
            className="inline-flex items-center justify-center gap-2 rounded-md bg-primary px-4 py-2 text-primary-foreground hover:bg-primary/90 transition"
          >
            <Home className="h-4 w-4" />
            Go Home
          </a>

          <button
            onClick={() => window.history.back()}
            className="inline-flex items-center justify-center gap-2 rounded-md border border-border px-4 py-2 hover:bg-muted transition"
          >
            <ArrowLeft className="h-4 w-4" />
            Go Back
          </button>
        </div>

        {/* Footer hint */}
        <p className="mt-8 text-xs text-muted-foreground">
          LogPulse routing system • check URL or navigation
        </p>
      </div>
    </div>
  );
};

export default NotFound;