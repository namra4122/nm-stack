// Renders every <pre class="mermaid"> on the page using Mermaid from a CDN.
// Diagram source stays as plain text in the HTML, so git diffs stay readable.
// Classic script (not a module) so it works when the page is opened from file://.
// If Mermaid can't load (offline), the diagram source text stays visible.
(function () {
  var s = document.createElement("script");
  s.src = "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js";
  s.onload = function () {
    window.mermaid.initialize({
      startOnLoad: false,
      theme: "base",
      themeVariables: {
        fontFamily: "'DM Mono', monospace",
        fontSize: "13px",
        primaryColor: "#ffffff",
        primaryBorderColor: "#315d50",
        primaryTextColor: "#20231f",
        lineColor: "#62675f",
        secondaryColor: "#e9eee8",
        tertiaryColor: "#f7f6f0",
        noteBkgColor: "#e9eee8",
        noteBorderColor: "#315d50",
      },
      flowchart: { curve: "basis", htmlLabels: true },
      sequence: { mirrorActors: false },
    });
    window.mermaid.run({ querySelector: "pre.mermaid" });
  };
  document.head.appendChild(s);
})();
