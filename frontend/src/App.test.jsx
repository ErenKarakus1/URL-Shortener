import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import App from "./App";

function jsonResponse(status, body) {
  return Promise.resolve({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  });
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe("URL shortener", () => {
  it("submits a long URL and displays the short link", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        jsonResponse(201, {
          shorturl: "abc123",
          longurl: "https://example.com",
        }),
      ),
    );
    render(<App />);

    fireEvent.change(screen.getByLabelText("Long URL"), {
      target: { value: "example.com" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Shorten URL" }));

    expect(await screen.findByText("http://localhost:3000/abc123")).toBeInTheDocument();
    expect(fetch).toHaveBeenCalledWith(
      "/api/shorten",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ longurl: "example.com" }),
      }),
    );
  });

  it("shows a useful message for an unknown short code", async () => {
    window.history.replaceState({}, "", "/missing");
    vi.stubGlobal(
      "fetch",
      vi.fn(() => jsonResponse(404, { message: "Short URL not found" })),
    );

    render(<App />);

    expect(await screen.findByText("This short link does not exist.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Return to main page" })).toHaveAttribute(
      "href",
      "/",
    );
  });

  it("shows a user-friendly message when the service cannot be reached", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.reject(new TypeError("Failed to fetch"))));
    render(<App />);

    fireEvent.change(screen.getByLabelText("Long URL"), {
      target: { value: "example.com" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Shorten URL" }));

    expect(
      await screen.findByText(
        "We couldn’t connect to the service. Please check your connection and try again.",
      ),
    ).toBeInTheDocument();
  });

  it("does not expose a malformed server response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        Promise.resolve({
          ok: false,
          status: 500,
          json: () => Promise.reject(new SyntaxError("Unexpected token")),
        }),
      ),
    );
    render(<App />);

    fireEvent.change(screen.getByLabelText("Long URL"), {
      target: { value: "example.com" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Shorten URL" }));

    expect(
      await screen.findByText(
        "Something went wrong on our side. Please try again in a moment.",
      ),
    ).toBeInTheDocument();
  });

  it("cancels the destination redirect and starts the return-home countdown", async () => {
    window.history.replaceState({}, "", "/abc123");
    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        jsonResponse(200, {
          longurl: "https://example.com/path",
        }),
      ),
    );

    render(<App />);

    expect(await screen.findByText("https://example.com/path")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));

    expect(screen.getByText("Taking you back to the main page.")).toBeInTheDocument();
    expect(screen.getByText("Returning home in 3 seconds.")).toBeInTheDocument();
    expect(screen.getByRole("progressbar")).toHaveAttribute("aria-valuemax", "3");
    expect(screen.queryByRole("button", { name: "Redirect now" })).not.toBeInTheDocument();
  });
});
