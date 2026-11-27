// Reusable quiz widget.
// Usage: <div class="quiz" data-src="quiz-id"></div> plus
// <script type="application/json" id="quiz-id">[{q, options:[...], answer:index, why}]</script>
(function () {
  function render(root) {
    const items = JSON.parse(document.getElementById(root.dataset.src).textContent);
    let answered = 0, correct = 0;
    const score = document.createElement('p');
    score.className = 'quiz-score';
    items.forEach((item, n) => {
      const box = document.createElement('div');
      box.className = 'quiz-item';
      box.innerHTML = `<p class="quiz-q"><span class="label">Q${n + 1}</span> ${item.q}</p>`;
      const why = document.createElement('p');
      why.className = 'quiz-why';
      item.options.forEach((opt, i) => {
        const b = document.createElement('button');
        b.type = 'button';
        b.textContent = opt;
        b.onclick = () => {
          if (box.dataset.done) return;
          box.dataset.done = '1';
          answered++;
          const ok = i === item.answer;
          if (ok) correct++;
          b.classList.add(ok ? 'right' : 'wrong');
          box.querySelectorAll('button')[item.answer].classList.add('right');
          why.innerHTML = (ok ? '✓ ' : '✗ ') + item.why;
          score.textContent = `${correct} / ${answered} correct` +
            (answered === items.length ? ' — revisit the misses tomorrow, without notes.' : '');
        };
        box.appendChild(b);
      });
      box.appendChild(why);
      root.appendChild(box);
    });
    root.appendChild(score);
  }
  document.querySelectorAll('.quiz').forEach(render);
})();
