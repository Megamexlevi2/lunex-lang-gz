
(function(){
  const drawer = document.querySelector('.drawer');
  const openBtn = document.querySelector('.hamburger');
  const closeBtn = document.querySelector('.drawer-close');

  function openDrawer(){
    if(!drawer || !openBtn) return;
    drawer.classList.add('open');
    drawer.setAttribute('aria-hidden','false');
    openBtn.setAttribute('aria-expanded','true');
  }
  function closeDrawer(){
    if(!drawer || !openBtn) return;
    drawer.classList.remove('open');
    drawer.setAttribute('aria-hidden','true');
    openBtn.setAttribute('aria-expanded','false');
  }
  if(openBtn) openBtn.addEventListener('click', openDrawer);
  if(closeBtn) closeBtn.addEventListener('click', closeDrawer);
  if(drawer) drawer.addEventListener('click', e => { if(e.target === drawer) closeDrawer(); });

  document.querySelectorAll('[data-copy]').forEach(btn=>{
    btn.addEventListener('click', async ()=>{
      const value = btn.getAttribute('data-copy') || '';
      try{
        await navigator.clipboard.writeText(value);
        const old = btn.textContent;
        btn.textContent = 'Copied';
        setTimeout(()=>btn.textContent = old, 1200);
      }catch{
        const old = btn.textContent;
        btn.textContent = 'Copy failed';
        setTimeout(()=>btn.textContent = old, 1200);
      }
    });
  });

  const editor = document.getElementById('playground-editor');
  const output = document.getElementById('playground-output');
  const count = document.getElementById('playground-count');
  if(editor && output && count){
    const samples = {
      hello: `val io = @import("std.io")\n\nfn main() {\n  io.log("Hello from NTL")\n}`,
      control: `val io = @import("std.io")\n\nfn main() {\n  var i = 0\n  while i < 5 {\n    io.log("tick", i)\n    i += 1\n  }\n}`,
      module: `val io = @import("std.io")\n\nfn greet(name) {\n  "Hello, " + name + "!"\n}\n\nfn main() {\n  io.log(greet("David"))\n}`
    };

    function sync(){
      output.textContent = editor.value;
      count.textContent = editor.value.length + ' chars';
      localStorage.setItem('ntl-playground', editor.value);
    }
    const saved = localStorage.getItem('ntl-playground');
    if(saved) editor.value = saved;
    sync();

    editor.addEventListener('input', sync);
    document.querySelectorAll('[data-sample]').forEach(btn=>{
      btn.addEventListener('click', ()=>{
        const sample = samples[btn.getAttribute('data-sample')];
        if(sample){ editor.value = sample; sync(); }
      });
    });
    const reset = document.getElementById('playground-reset');
    if(reset) reset.addEventListener('click', ()=>{ editor.value = samples.hello; sync(); });
    const copy = document.getElementById('playground-copy');
    if(copy) copy.addEventListener('click', async ()=>{
      try{
        await navigator.clipboard.writeText(editor.value);
        const old = copy.textContent;
        copy.textContent = 'Copied';
        setTimeout(()=>copy.textContent = old, 1200);
      }catch{
        const old = copy.textContent;
        copy.textContent = 'No clipboard';
        setTimeout(()=>copy.textContent = old, 1200);
      }
    });
  }

  const tocLinks = document.querySelectorAll('.toc a');
  const sections = document.querySelectorAll('.doc-content section[id]');
  if(tocLinks.length && sections.length){
    const map = new Map(Array.from(tocLinks).map(a => [a.getAttribute('href').slice(1), a]));
    const observer = new IntersectionObserver(entries=>{
      entries.forEach(entry=>{
        if(entry.isIntersecting){
          tocLinks.forEach(a=>a.classList.remove('active'));
          const a = map.get(entry.target.id);
          if(a) a.classList.add('active');
        }
      });
    }, {rootMargin:'-30% 0px -60% 0px', threshold:0.1});
    sections.forEach(sec => observer.observe(sec));
  }
})();
