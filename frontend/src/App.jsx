import { useEffect, useState } from "react";

const apiBaseURL = (import.meta.env.VITE_API_URL || "/api").replace(/\/$/, "");
const shortURLBase = (
  import.meta.env.VITE_SHORT_URL_BASE || window.location.origin
).replace(/\/$/, "");

function buildShortURL(code) {
  return `${shortURLBase}/${code}`;
}

function HomePage() {
  const [longURL, setLongURL] = useState("");
  const [shortURL, setShortURL] = useState("");
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [copied, setCopied] = useState(false);

  async function handleSubmit(event) {
    event.preventDefault();
    setError("");
    setShortURL("");
    setCopied(false);
    setIsLoading(true);

    try {
      const response = await fetch(`${apiBaseURL}/shorten`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ longurl: longURL }),
      });
      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.message || "Unable to shorten this URL.");
      }

      setShortURL(buildShortURL(data.shorturl));
    } catch (requestError) {
      setError(
        requestError instanceof TypeError
          ? "The server is unavailable. Make sure the backend is running."
          : requestError.message,
      );
    } finally {
      setIsLoading(false);
    }
  }

  async function copyShortURL() {
    await navigator.clipboard.writeText(shortURL);
    setCopied(true);
  }

  return (
    <main className="page-shell">
      <section className="shortener-card" aria-labelledby="page-title">
        <div className="brand-mark" aria-hidden="true">
          ↗
        </div>
        <p className="eyebrow">Simple URL shortener</p>
        <h1 id="page-title">Make long links easier to share.</h1>
        <p className="intro">
          Paste a valid web address and get a permanent short link in seconds.
        </p>

        <form onSubmit={handleSubmit}>
          <label htmlFor="long-url">Long URL</label>
          <div className="input-row">
            <input
              id="long-url"
              type="text"
              inputMode="url"
              value={longURL}
              onChange={(event) => setLongURL(event.target.value)}
              placeholder="example.com/a/very/long/link"
              autoComplete="url"
              required
            />
            <button type="submit" disabled={isLoading}>
              {isLoading ? "Shortening…" : "Shorten URL"}
            </button>
          </div>
        </form>

        {error && (
          <p className="message error-message" role="alert">
            {error}
          </p>
        )}

        {shortURL && (
          <div className="result" aria-live="polite">
            <div>
              <span>Your short link</span>
              <a href={shortURL}>{shortURL}</a>
            </div>
            <button className="copy-button" type="button" onClick={copyShortURL}>
              {copied ? "Copied" : "Copy"}
            </button>
          </div>
        )}
      </section>

      <p className="privacy-note">No accounts. No analytics. Just shorter links.</p>
    </main>
  );
}

function RedirectPage({ shortCode }) {
  const [destination, setDestination] = useState("");
  const [error, setError] = useState("");
  const [phase, setPhase] = useState("loading");
  const [remaining, setRemaining] = useState(5);
  const total = phase === "returning" ? 3 : 5;

  useEffect(() => {
    const controller = new AbortController();

    async function resolveURL() {
      try {
        const response = await fetch(
          `${apiBaseURL}/resolve/${encodeURIComponent(shortCode)}`,
          { signal: controller.signal },
        );
        const data = await response.json();

        if (!response.ok) {
          throw new Error(
            response.status === 404
              ? "This short link does not exist."
              : data.message || "Unable to resolve this short link.",
          );
        }

        setDestination(data.longurl);
        setPhase("redirecting");
      } catch (requestError) {
        if (requestError.name !== "AbortError") {
          setError(
            requestError instanceof TypeError
              ? "The server is unavailable. Make sure the backend is running."
              : requestError.message,
          );
          setPhase("error");
        }
      }
    }

    resolveURL();
    return () => controller.abort();
  }, [shortCode]);

  useEffect(() => {
    if (phase !== "redirecting" && phase !== "returning") {
      return undefined;
    }

    const timer = window.setInterval(() => {
      setRemaining((current) => {
        if (current <= 1) {
          window.clearInterval(timer);
          window.location.assign(phase === "redirecting" ? destination : "/");
          return 0;
        }
        return current - 1;
      });
    }, 1000);

    return () => window.clearInterval(timer);
  }, [destination, phase]);

  function cancelRedirect() {
    setPhase("returning");
    setRemaining(3);
  }

  const progress = Math.max(0, (remaining / total) * 100);

  return (
    <main className="page-shell">
      <section className="shortener-card redirect-card" aria-labelledby="redirect-title">
        <div className="brand-mark redirect-mark" aria-hidden="true">
          ↗
        </div>

        {phase === "loading" && (
          <>
            <p className="eyebrow">Checking link</p>
            <h1 id="redirect-title">Finding your destination…</h1>
          </>
        )}

        {phase === "error" && (
          <>
            <p className="eyebrow">Link unavailable</p>
            <h1 id="redirect-title">We couldn’t open this link.</h1>
            <p className="intro" role="alert">
              {error}
            </p>
            <a className="home-link" href="/">
              Return to main page
            </a>
          </>
        )}

        {(phase === "redirecting" || phase === "returning") && (
          <>
            <p className="eyebrow">
              {phase === "returning" ? "Redirect cancelled" : "Destination found"}
            </p>
            <h1 id="redirect-title">
              {phase === "returning"
                ? "Taking you back to the main page."
                : "You’re leaving Shortly."}
            </h1>
            <p className="intro redirect-status" aria-live="polite">
              {phase === "returning"
                ? `Returning home in ${remaining} seconds.`
                : `Redirecting in ${remaining} seconds to`}
            </p>

            {phase === "redirecting" && (
              <span className="destination" title={destination}>
                {destination}
              </span>
            )}

            <div
              className="countdown-track"
              role="progressbar"
              aria-label="Time remaining"
              aria-valuemin="0"
              aria-valuemax={total}
              aria-valuenow={remaining}
            >
              <div className="countdown-bar" style={{ width: `${progress}%` }} />
            </div>

            {phase === "redirecting" && (
              <div className="redirect-actions">
                <button
                  className="redirect-now"
                  type="button"
                  onClick={() => window.location.assign(destination)}
                >
                  Redirect now
                </button>
                <button className="cancel-redirect" type="button" onClick={cancelRedirect}>
                  Cancel
                </button>
              </div>
            )}
          </>
        )}
      </section>
    </main>
  );
}

export default function App() {
  const shortCode = window.location.pathname.replace(/^\/+|\/+$/g, "");

  return shortCode ? <RedirectPage shortCode={shortCode} /> : <HomePage />;
}
