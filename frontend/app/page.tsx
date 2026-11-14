const productPrinciples = [
  {
    title: "Run where the repository lives",
    description:
      "The Go service runs on the same machine as the repository you choose. Source code and local paths stay local.",
  },
  {
    title: "Inspect without execution",
    description:
      "Static analysis and read-only Git history inspection are the only planned inputs—repository code is never run.",
  },
  {
    title: "Explain every priority",
    description:
      "Future reports will show complexity, change history, and score contributions so review decisions remain understandable.",
  },
];

export default function HomePage() {
  return (
    <main className="site-shell">
      <section className="hero" aria-labelledby="page-title">
        <p className="eyebrow">Local-first code intelligence</p>
        <h1 id="page-title">Understand repository risk locally.</h1>
        <p className="hero-copy">
          Code Risk Analyzer will help engineering teams prioritize review and
          refactoring using static complexity signals and Git history. No source
          code is uploaded.
        </p>
        <div className="local-boundary" role="note">
          <span aria-hidden="true">●</span>
          <p>
            This UI calls only relative API routes. The local Next.js server
            proxies them to the Go service running beside your repository.
          </p>
        </div>
      </section>

      <section className="principles" aria-labelledby="principles-title">
        <div className="section-heading">
          <p className="eyebrow">Product boundary</p>
          <h2 id="principles-title">Built for trustworthy technical decisions.</h2>
        </div>
        <ul className="principle-grid" role="list">
          {productPrinciples.map((principle, index) => (
            <li className="principle-card" key={principle.title}>
              <span className="principle-number" aria-hidden="true">
                0{index + 1}
              </span>
              <h3>{principle.title}</h3>
              <p>{principle.description}</p>
            </li>
          ))}
        </ul>
      </section>

      <section className="next-step" aria-labelledby="next-step-title">
        <div>
          <p className="eyebrow">Foundation status</p>
          <h2 id="next-step-title">Analysis workspace is coming next.</h2>
          <p>
            The runnable frontend and local API foundation are in place. The
            repository path form, static analysis, Git history metrics, risk
            scoring, and dashboard will appear only after each capability is
            implemented and tested.
          </p>
        </div>
        <a className="health-link" href="/api/v1/health">
          Check local API health
          <span aria-hidden="true">→</span>
        </a>
      </section>
    </main>
  );
}
