import { useState } from "react";

const apiBaseURL = (import.meta.env.VITE_API_URL || "/api").replace(/\/$/, "");

function buildShortURL(code) {
  if (apiBaseURL.startsWith("http://") || apiBaseURL.startsWith("https://")) {
    return `${apiBaseURL}/${code}`;
  }
  return `${window.location.origin}${apiBaseURL}/${code}`;
}

export default function App() {
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
              type="url"
              value={longURL}
              onChange={(event) => setLongURL(event.target.value)}
              placeholder="https://example.com/a/very/long/link"
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
              <a href={shortURL} target="_blank" rel="noreferrer">
                {shortURL}
              </a>
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
