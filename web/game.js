'use strict';
/* =====================================================================
   OLD MAID - A Victorian Parlor Game                (c) 2026 Ironite
   ===================================================================== */
const VERSION = '1.0.0';
(async function () {
  const $ = s => document.querySelector(s);
  const sleepRaw = ms => new Promise(r => setTimeout(r, ms));
  const rand = n => Math.floor(Math.random() * n);
  const pick = a => a[rand(a.length)];
  const TEST = /[?&]test/.test(location.search);

  /* ---------------- saved settings & record ---------------- */
  const DEFAULTS = { settings: { music: true, sfx: true, vol: 60, opponents: 3, speed: 'normal' }, stats: { played: 0, safe: 0, maid: 0, first: 0, streak: 0, best: 0 } };
  let save = JSON.parse(JSON.stringify(DEFAULTS)), saveTimer = null;
  async function loadSave() {
    let data = null;
    try { const r = await fetch('api/save', { cache: 'no-store' }); if (r.ok) data = await r.json(); } catch (e) { }
    if (!data) try { data = JSON.parse(localStorage.getItem('oldmaid.save') || 'null'); } catch (e) { }
    if (data) { Object.assign(save.settings, data.settings || {}); Object.assign(save.stats, data.stats || {}); }
  }
  function writeSave() {
    const body = JSON.stringify(save);
    try { localStorage.setItem('oldmaid.save', body); } catch (e) { }
    fetch('api/save', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body, keepalive: true }).catch(() => { });
  }
  function persist(now) { clearTimeout(saveTimer); if (now) writeSave(); else saveTimer = setTimeout(writeSave, 250); }
  window.addEventListener('pagehide', () => { clearTimeout(saveTimer); writeSave(); });

  /* ---------------- pacing ---------------- */
  const PACE = { relaxed: 1.35, normal: 1, brisk: 0.62 };
  let pace = 1, rush = 1;                 // rush speeds things up once you're out
  const T = k => ({ deal: 55, move: 430, flip: 380, pair: 520, think: 700, tempt: 380, quip: 1900 }[k]) * pace * rush * (TEST ? 0.08 : 1);

  /* ---------------- the company ---------------- */
  const PEOPLE = {
    agatha: { name: 'Lady Agatha', cam: 'agatha',
      pair: ['Splendid.', 'As expected.', 'Quite so.', 'Naturally.', 'A matched set.'],
      draw: ['Let me see…', 'Hmm. This one.', 'Do hold still, dear.'],
      out: ['I shall retire to the drawing room.', 'That will be all, thank you.'],
      maid: ['Well! I never!', 'How utterly vexing.'],
      took: ['Thank you, dear.', 'Hmph.'] },
    colonel: { name: 'Colonel Bramble', cam: 'colonel',
      pair: ['Ha! Another brace!', 'Capital!', 'Bully for me!', 'Splendid shot!', 'Right-ho!'],
      draw: ['Eeny, meeny…', 'Steady now…', 'Into the breach!'],
      out: ['Strategic withdrawal, what!', 'Mission accomplished!'],
      maid: ['Ambushed, by Jove!', 'Confound it!'],
      took: ['Hm! Interesting.', 'Ha! Much obliged.'] },
    pettigrew: { name: 'Miss Pettigrew', cam: 'pettigrew',
      pair: ['Oh, how lovely!', 'A match!', 'Delightful!', 'Oh, goody!', 'Hee hee!'],
      draw: ['Oh, which one?', 'This one looks lucky!', 'Ooh, eenie…'],
      out: ["I'm out! Oh, what fun!", 'All done! Hooray!'],
      maid: ['Oh, bother!', 'Oh no, not her!'],
      took: ['Oh! Thank you!', 'Ooh.'] }
  };
  const SEATING = { 1: ['N'], 2: ['W', 'E'], 3: ['W', 'N', 'E'] };
  const WHO = { 1: ['colonel'], 2: ['agatha', 'pettigrew'], 3: ['agatha', 'colonel', 'pettigrew'] };

  /* ---------------- fonts, art ---------------- */
  try { await Promise.all([document.fonts.load('40px "IM Fell English"'), document.fonts.load('40px "IM Fell English SC"'), document.fonts.load('italic 20px "IM Fell English"')]); } catch (e) { }
  await loadSave();
  const backURL = Art.back().toDataURL('image/png');
  const DECK = [];
  for (const s of ['S', 'H', 'D', 'C']) for (let r = 1; r <= 13; r++) {
    if (r === 12 && s === 'C') continue;                     // one queen leaves the deck...
    DECK.push({ id: s + r, r, s, red: s === 'H' || s === 'D', maid: r === 12 && s === 'S' });  // ...so her black sister has no partner
  }
  for (const c of DECK) {
    const el = document.createElement('div'); el.className = 'card down';
    el.innerHTML = `<div class="lift"><div class="inner"><div class="face front"></div><div class="face back" style="background-image:url(${backURL})"></div></div></div>`;
    el.querySelector('.front').appendChild(Art.face(c));
    el.addEventListener('click', () => onCardClick(c));
    el.addEventListener('mouseenter', () => { if (el.classList.contains('pick')) Sound.play('hover'); });
    c.el = el; c.up = false;
  }
  // felt texture
  (() => { const cv = document.createElement('canvas'); cv.width = cv.height = 160; const c = cv.getContext('2d');
    for (let i = 0; i < 2600; i++) { const v = Math.random(); c.fillStyle = v > .5 ? `rgba(255,255,255,${Math.random() * .05})` : `rgba(0,0,0,${Math.random() * .08})`; c.fillRect(Math.random() * 160, Math.random() * 160, 1, 1 + Math.random() * 2); }
    document.documentElement.style.setProperty('--felt-noise', `url(${cv.toDataURL()})`); })();
  // icon for the window
  (() => { const l = document.querySelector('link[rel=icon]'); l.href = Art.icon(64).toDataURL(); })();

  /* ---------------- state ---------------- */
  let players = [], turn = 0, gameId = 0, discards = 0, cw = 100, ch = 140, Wd = 0, Ht = 0, busyClick = null, finished = false, started = false;
  class Cancel extends Error { }
  let liveId = 0;
  const sleep = async ms => { const id = liveId; await sleepRaw(ms); if (id !== gameId) throw new Cancel(); };

  /* ---------------- layout ---------------- */
  const felt = $('#felt'), cardsEl = $('#cards'), seatsEl = $('#seats');
  function measure() {
    Wd = felt.clientWidth; Ht = felt.clientHeight;
    ch = Math.max(92, Math.min(Ht * 0.2, Wd * 0.125, 210)); cw = ch * (Art.CW / Art.CH);
    felt.style.setProperty('--cw', cw + 'px'); felt.style.setProperty('--ch', ch + 'px');
    const d = $('#doily'), ds = ch * 1.9; d.style.width = d.style.height = ds + 'px'; d.style.left = (Wd / 2 - ds / 2) + 'px'; d.style.top = (pileY() - ds / 2) + 'px';
    for (const cv of [$('#dust'), $('#fx')]) { cv.width = Wd; cv.height = Ht; }
  }
  const pileY = () => Ht * 0.47;
  function seatAnchor(seat) {
    if (seat === 'S') return { x: Wd / 2, y: Ht - ch * 0.62 - 6 };
    if (seat === 'N') return { x: Wd / 2, y: ch * 0.62 + 8 };
    if (seat === 'W') return { x: ch * 0.62 + 10, y: Ht * 0.41 };
    return { x: Wd - ch * 0.62 - 10, y: Ht * 0.41 };
  }
  function handPos(p, i, n) {
    const a = seatAnchor(p.seat), k = i - (n - 1) / 2;
    if (p.seat === 'S') {
      const sp = n > 1 ? Math.min(cw * 0.66, (Wd * 0.62 - cw) / (n - 1)) : 0, fan = Math.min(2.6, 34 / Math.max(n, 1));
      return { x: a.x + k * sp, y: a.y + Math.abs(k) * Math.abs(k) * 0.9, r: k * fan };
    }
    if (p.seat === 'N') {
      const sp = n > 1 ? Math.min(cw * 0.36, (Wd * 0.30) / (n - 1)) : 0;
      return { x: a.x - k * sp, y: a.y - Math.abs(k) * Math.abs(k) * 0.5, r: 180 - k * Math.min(2.2, 26 / n) };
    }
    const sp = n > 1 ? Math.min(cw * 0.34, (Ht * 0.34) / (n - 1)) : 0, dir = p.seat === 'W' ? 1 : -1;
    return { x: a.x - dir * Math.abs(k) * Math.abs(k) * 0.5, y: a.y + dir * k * sp, r: dir * 90 + dir * k * Math.min(2.2, 26 / n) };
  }
  function place(c, x, y, r, z, scale = 1) {
    c.x = x; c.y = y; c.rot = r;
    c.el.style.transform = `translate(${x - cw / 2}px, ${y - ch / 2}px) rotate(${r}deg) scale(${scale})`;
    if (z !== undefined) c.el.style.zIndex = z;
  }
  function flip(c, up) { c.up = up; c.el.classList.toggle('down', !up); }
  function layoutHand(p) { p.hand.forEach((c, i) => { const q = handPos(p, i, p.hand.length); place(c, q.x, q.y, q.r, 100 + i); }); }
  function layoutAll() { for (const p of players) layoutHand(p); layoutSeats(); }
  function sortHuman(p) { p.hand.sort((a, b) => a.r - b.r || (a.red - b.red) || a.s.localeCompare(b.s)); }

  function layoutSeats() {
    for (const p of players) {
      if (!p.seatEl) continue;
      let x, y; const a = seatAnchor(p.seat);
      if (p.seat === 'N') { x = Wd / 2 - Wd * 0.15 - cw * 0.5 - 130; y = a.y - 4; }
      else if (p.seat === 'W') { x = a.x + 48; y = a.y + Ht * 0.17 + cw * 0.5 + 58; }
      else if (p.seat === 'E') { x = a.x - 48; y = a.y + Ht * 0.17 + cw * 0.5 + 58; }
      else { x = Wd / 2 - Wd * 0.31 - cw * 0.5 - 70; y = Ht - ch * 0.62; }
      p.seatEl.style.left = Math.max(110, Math.min(Wd - 110, x)) + 'px'; p.seatEl.style.top = y + 'px';
    }
    const pr = $('#prompt'); pr.style.top = (Ht - ch * 1.32 - 34) + 'px';
  }
  function seatInfo(p) {
    if (!p.seatEl) return;
    const n = p.hand.length;
    p.seatEl.querySelector('.ct').textContent = p.out ? `Safe — ${ordinal(p.place)} out` : `${n} card${n === 1 ? '' : 's'}`;
    p.seatEl.classList.toggle('out', p.out);
  }
  const ordinal = n => n + (['th', 'st', 'nd', 'rd'][(n % 100 > 10 && n % 100 < 14) ? 0 : n % 10] || 'th');
  function setTurn(p) { for (const q of players) q.seatEl && q.seatEl.classList.toggle('turn', q === p); }
  function prompt(text) { const el = $('#prompt'); el.textContent = text || ''; el.style.opacity = text ? 1 : 0; }
  let bannerTimer = null;
  function banner(html, ms = 1800) { const b = $('#banner'); b.innerHTML = html; b.classList.add('show'); clearTimeout(bannerTimer); bannerTimer = setTimeout(() => b.classList.remove('show'), ms); }
  function say(p, kind, chance = 1) {
    if (!p.person || Math.random() > chance) return;
    const b = p.seatEl.querySelector('.bubble'); b.textContent = pick(p.person[kind]); b.classList.add('show');
    clearTimeout(p.sayT); p.sayT = setTimeout(() => b.classList.remove('show'), T('quip') + 400);
  }

  /* ---------------- effects ---------------- */
  const fxc = $('#fx').getContext('2d'), dustc = $('#dust').getContext('2d');
  const parts = [], motes = [];
  for (let i = 0; i < 46; i++) motes.push({ x: Math.random(), y: Math.random(), r: 0.6 + Math.random() * 1.8, vx: (Math.random() - .5) * 0.00006, vy: -0.00002 - Math.random() * 0.00005, p: Math.random() * 6 });
  function burst(x, y, kind, n) {
    for (let i = 0; i < n; i++) {
      if (kind === 'spark') { const a = Math.random() * Math.PI * 2, v = 1.5 + Math.random() * 4; parts.push({ k: 'spark', x, y, vx: Math.cos(a) * v, vy: Math.sin(a) * v - 1, life: 1, d: 0.018 + Math.random() * 0.02, s: 2 + Math.random() * 3, rot: Math.random() * 6 }); }
      else if (kind === 'puff') { // a ring of smoke rolling outward from the card's edge
        const a = Math.random() * Math.PI * 2, v = 0.8 + Math.random() * 1.8, rr = (n > 24 ? cw * 1.25 : cw * 0.5);
        parts.push({ k: 'puff', x: x + Math.cos(a) * rr, y: y + Math.sin(a) * rr * 1.3, vx: Math.cos(a) * v, vy: Math.sin(a) * v - 0.4, life: 1, d: 0.011 + Math.random() * 0.008, s: 12 + Math.random() * 18, col: Math.random() < .5 ? '120,90,140' : '90,80,90' }); }
      else if (kind === 'petal') { parts.push({ k: 'petal', x: Math.random() * Wd, y: -20 - Math.random() * Ht * 0.6, vx: (Math.random() - .5) * 1.2, vy: 1 + Math.random() * 1.6, life: 1, d: 0.0025, s: 6 + Math.random() * 6, rot: Math.random() * 6, vr: (Math.random() - .5) * 0.1, col: pick(['#c8324a', '#e46b7a', '#f2a3b0', '#fbe3e8', '#d9b45a']) }); }
    }
  }
  let lastT = performance.now();
  function frame(t) {
    const dt = Math.min(50, t - lastT); lastT = t;
    dustc.clearRect(0, 0, Wd, Ht);
    for (const m of motes) {
      m.x += m.vx * dt; m.y += m.vy * dt; m.p += dt * 0.0012; if (m.y < -0.02) { m.y = 1.02; m.x = Math.random(); } if (m.x < 0) m.x = 1; if (m.x > 1) m.x = 0;
      dustc.fillStyle = `rgba(255,230,180,${0.12 + Math.sin(m.p) * 0.08})`; dustc.beginPath(); dustc.arc(m.x * Wd, m.y * Ht, m.r, 0, 6.283); dustc.fill();
    }
    fxc.clearRect(0, 0, Wd, Ht);
    for (let i = parts.length - 1; i >= 0; i--) {
      const p = parts[i]; const f = dt / 16.7;
      p.x += p.vx * f; p.y += p.vy * f; p.life -= p.d * f;
      if (p.k === 'spark') { p.vy += 0.05 * f; p.vx *= 0.98; fxc.save(); fxc.translate(p.x, p.y); fxc.rotate(p.rot += 0.1 * f); fxc.globalAlpha = Math.max(0, p.life); fxc.fillStyle = '#ffe9a8'; fxc.beginPath(); for (let j = 0; j < 4; j++) { fxc.rotate(Math.PI / 2); fxc.moveTo(0, 0); fxc.lineTo(p.s * 0.5, p.s * 0.5); fxc.lineTo(0, p.s * 2.2); fxc.lineTo(-p.s * 0.5, p.s * 0.5); } fxc.fill(); fxc.restore(); }
      else if (p.k === 'puff') { p.s += 0.6 * f; fxc.globalAlpha = Math.max(0, p.life) * 0.34; fxc.fillStyle = `rgb(${p.col})`; fxc.beginPath(); fxc.arc(p.x, p.y, p.s, 0, 6.283); fxc.fill(); fxc.globalAlpha = 1; }
      else if (p.k === 'petal') { p.vx += Math.sin(t / 400 + i) * 0.02; p.rot += p.vr * f; fxc.save(); fxc.translate(p.x, p.y); fxc.rotate(p.rot); fxc.globalAlpha = Math.min(1, p.life * 2); fxc.fillStyle = p.col; fxc.beginPath(); fxc.ellipse(0, 0, p.s, p.s * 0.55, 0, 0, 6.283); fxc.fill(); fxc.restore(); if (p.y > Ht + 20) p.life = 0; }
      if (p.life <= 0) parts.splice(i, 1);
    }
    requestAnimationFrame(frame);
  }
  requestAnimationFrame(frame);

  /* ---------------- game flow ---------------- */
  const active = () => players.filter(p => !p.out);
  function nextActive(i) { for (let k = 1; k <= players.length; k++) { const j = (i + k) % players.length; if (!players[j].out) return j; } return -1; }
  let place_ = 1;

  function makeTable() {
    seatsEl.innerHTML = '';
    const n = save.settings.opponents, seats = SEATING[n], who = WHO[n];
    players = [{ seat: 'S', name: 'You', human: true, hand: [], out: false }];
    seats.forEach((s, i) => players.push({ seat: s, name: PEOPLE[who[i]].name, person: PEOPLE[who[i]], hand: [], out: false }));
    for (const p of players) {
      const el = document.createElement('div'); el.className = 'seat' + (p.human ? ' me' : '');
      el.innerHTML = `<div class="bubble"></div>` + (p.human ? '' : '') + `<div class="who"><div class="nm"></div><div class="ct"></div></div>`;
      if (!p.human) { const cam = Art.cameo(p.person.cam, 152); cam.className = 'cam'; el.prepend(cam); }
      el.querySelector('.nm').textContent = p.human ? 'You' : p.name;
      seatsEl.appendChild(el); p.seatEl = el;
    }
  }

  async function newGame() {
    gameId++; liveId = gameId; finished = false; started = true; rush = 1; pace = PACE[save.settings.speed] || 1; place_ = 1; discards = 0; busyClick = null;
    $('#result').classList.add('hidden');
    makeTable(); measure(); layoutSeats();
    cardsEl.innerHTML = '';
    const deck = DECK.slice();
    for (let i = deck.length - 1; i > 0; i--) { const j = rand(i + 1); [deck[i], deck[j]] = [deck[j], deck[i]]; }
    deck.forEach((c, i) => { c.el.className = 'card down instant'; c.up = false; place(c, Wd / 2 + (i - 25) * 0.25, pileY() - i * 0.3, (Math.random() - .5) * 4, 20 + i); cardsEl.appendChild(c.el); });
    for (const p of players) seatInfo(p);
    prompt('Shuffling…');
    try {
      await sleep(60); deck.forEach(c => c.el.classList.remove('instant'));
      await shuffleAnim(deck);
      // deal
      prompt('Dealing…');
      let k = 0;
      for (const c of deck) {
        const p = players[k++ % players.length];
        p.hand.push(c); c.el.classList.add('fast');
        if (p.human) { sortHuman(p); flip(c, true); }
        layoutHand(p); seatInfo(p); if (k % 2) Sound.play('deal');
        await sleep(T('deal'));
      }
      deck.forEach(c => c.el.classList.remove('fast'));
      await sleep(T('move'));
      // everyone lays down their pairs
      prompt('Laying down pairs…');
      for (const p of players) { await layDownAll(p); }
      for (const p of players) if (!p.hand.length && !p.out) markOut(p);
      turn = 0;
      await play();
    } catch (e) { if (!(e instanceof Cancel)) console.error(e); }
  }

  async function shuffleAnim(deck) {
    Sound.play('shuffle');
    for (let rep = 0; rep < 3; rep++) {
      deck.forEach((c, i) => { const side = i % 2 ? 1 : -1; place(c, Wd / 2 + side * cw * 0.55, pileY() - i * 0.3, side * 8); });
      await sleep(T('flip') * 0.55);
      deck.forEach((c, i) => place(c, Wd / 2 + (Math.random() - .5) * 3, pileY() - i * 0.3, (Math.random() - .5) * 4, 20 + i));
      await sleep(T('flip') * 0.55);
    }
  }

  function findPair(p, c) { return p.hand.find(x => x !== c && x.r === c.r && x.red === c.red); }
  function discardSpot() {
    const a = Math.random() * Math.PI * 2, d = Math.sqrt(Math.random()) * ch * 0.34;
    return { x: Wd / 2 + Math.cos(a) * d, y: pileY() + Math.sin(a) * d * 0.8, r: (Math.random() - .5) * 70 };
  }
  async function discardPair(p, a, b, quick) {
    p.hand = p.hand.filter(x => x !== a && x !== b);
    for (const c of [a, b]) { c.el.style.zIndex = 900; if (!c.up) flip(c, true); }
    if (!quick) { a.el.classList.add('glow'); b.el.classList.add('glow'); await sleep(T('flip') * 0.9); }
    const s = discardSpot();
    place(a, s.x - cw * 0.14, s.y, s.r - 8, 30 + discards++, 0.8); place(b, s.x + cw * 0.14, s.y, s.r + 8, 30 + discards++, 0.8);
    layoutHand(p); seatInfo(p);
    await sleep(T('move') * 0.85);
    a.el.classList.remove('glow'); b.el.classList.remove('glow');
    Sound.play('pair'); burst(s.x, s.y, 'spark', quick ? 8 : 16);
  }
  async function layDownAll(p) {
    let found = true, any = false;
    while (found) {
      found = false;
      for (const c of p.hand) { const m = findPair(p, c); if (m) { found = true; any = true; discardPair(p, c, m, true).catch(() => { }); await sleep(T('pair') * 0.45); break; } }
    }
    if (any) await sleep(T('move'));
    seatInfo(p);
  }
  function markOut(p) {
    p.out = true; p.place = place_++; seatInfo(p);
    if (p.human) { banner(`You're out!<small>${p.place === 1 ? 'First to finish — splendid!' : 'Safe and sound.'}</small>`, 2400); Sound.play('out'); rush = 0.55; }
    else { say(p, 'out'); Sound.play('out'); }
  }

  async function play() {
    while (true) {
      if (active().length <= 1) return endGame();
      const p = players[turn];
      if (!p.out) {
        const gi = nextActive(turn), g = players[gi];
        setTurn(p);
        const c = p.human ? await humanPick(g) : await aiPick(p, g);
        await takeCard(p, g, c);
      }
      const n = nextActive(turn); if (n < 0) return endGame(); turn = n;
    }
  }

  function humanPick(g) {
    prompt(`Your turn — draw a card from ${g.name}`);
    g.hand.forEach(c => c.el.classList.add('pick'));
    return new Promise((resolve) => { busyClick = { g, resolve }; });
  }
  function onCardClick(c) {
    if (!busyClick || !busyClick.g.hand.includes(c)) return;
    const { g, resolve } = busyClick; busyClick = null;
    g.hand.forEach(x => x.el.classList.remove('pick'));
    prompt(''); Sound.unlock(); resolve(c);
  }
  async function aiPick(p, g) {
    prompt(g.human ? `${p.name} is choosing one of your cards…` : `${p.name} draws from ${g.name}`);
    if (g.human) {
      say(p, 'draw', 0.45);
      let last = null;
      for (let i = 0, n = 2 + rand(2); i < n; i++) {
        if (last) last.el.classList.remove('tempt');
        last = pick(g.hand); last.el.classList.add('tempt'); Sound.play('hover');
        await sleep(T('tempt'));
      }
      if (last) last.el.classList.remove('tempt');
      const choice = Math.random() < 0.5 && last ? last : pick(g.hand);
      choice.el.classList.add('tempt'); await sleep(T('tempt') * 0.6); choice.el.classList.remove('tempt');
      return choice;
    }
    await sleep(T('think') * (0.6 + Math.random() * 0.6));
    return pick(g.hand);
  }

  async function takeCard(p, g, c) {
    g.hand = g.hand.filter(x => x !== c);
    // into the new hand: humans keep theirs sorted, the others tuck it in anywhere
    if (p.human) { p.hand.push(c); sortHuman(p); } else p.hand.splice(rand(p.hand.length + 1), 0, c);
    if (p.human) flip(c, true); else if (c.up) flip(c, false);
    c.el.style.zIndex = 950; Sound.play('slide');
    layoutHand(g); layoutHand(p); seatInfo(g); seatInfo(p);
    if (p.human) c.el.classList.add('new');
    await sleep(T('move'));
    if (p.human) Sound.play('flip');
    // the Old Maid changes hands
    if (c.maid && p.human) { Sound.play('maid'); burst(c.x, c.y, 'puff', 18); banner('Oh dear!<small>You have drawn the Old Maid.</small>', 2200); await sleep(T('pair') * 1.4); }
    else if (c.maid && g.human) { Sound.play('phew'); banner(`Phew!<small>${p.name} took the Old Maid.</small>`, 2000); say(p, 'took', 0.6); }
    else if (g.human && !p.human) say(p, 'took', 0.15);
    const m = findPair(p, c);
    if (m) { await discardPair(p, c, m, false); say(p, 'pair', 0.4); }
    c.el.classList.remove('new');
    if (!g.hand.length && !g.out) markOut(g);
    if (!p.hand.length && !p.out) markOut(p);
    await sleep(T('move') * 0.35);
  }

  async function endGame() {
    if (finished) return; finished = true; setTurn(null); prompt('');
    const loser = active()[0] || players.find(p => p.hand.some(c => c.maid));
    const maid = DECK.find(c => c.maid);
    loser.out = true; loser.place = players.length; loser.maidHolder = true; seatInfo(loser);
    if (loser.seatEl) loser.seatEl.querySelector('.ct').textContent = 'The Old Maid!';
    // the reveal
    loser.hand = [];
    maid.el.style.zIndex = 3000; flip(maid, true);
    place(maid, Wd / 2, pileY(), 0, 3000, 2.2);
    await sleep(T('move') * 1.4);
    burst(Wd / 2, pileY(), 'puff', 34);
    const human = players[0];
    const st = save.stats; st.played++;
    if (loser.human) { st.maid++; st.streak = 0; Sound.play('lose'); }
    else { st.safe++; st.streak++; st.best = Math.max(st.best, st.streak); if (human.place === 1) st.first++; Sound.play('win'); burst(0, 0, 'petal', human.place === 1 ? 140 : 60); say(loser, 'maid'); }
    persist(true); showStats();
    await sleep(1600 * (TEST ? 0.1 : 1));
    // results card
    $('#rTitle').textContent = loser.human ? "You're the Old Maid!" : (human.place === 1 ? 'Splendid!' : 'Safe and Sound!');
    $('#rSub').textContent = loser.human ? 'Better luck at the next deal, dear.' : `${loser.name} is left holding the Old Maid.`;
    const order = players.slice().sort((a, b) => a.place - b.place);
    $('#rPlaces').innerHTML = order.map(p => `<li class="${p.maidHolder ? 'maid' : ''}">${p.human ? '<b>You</b>' : p.name}${p.maidHolder ? ' — the Old Maid' : p.place === 1 ? ' — first out' : ''}</li>`).join('');
    $('#rRecord').textContent = recordText();
    $('#result').classList.remove('hidden');
  }

  /* ---------------- UI ---------------- */
  function recordText() {
    const s = save.stats; if (!s.played) return 'No hands played yet.';
    return `Hands played: ${s.played} · Safe: ${s.safe} · Old Maid: ${s.maid} · First out: ${s.first} · Best safe streak: ${s.best}`;
  }
  function showStats() { $('#sPlayed').textContent = save.stats.played; $('#sSafe').textContent = save.stats.safe; $('#sMaid').textContent = save.stats.maid; $('#record').textContent = recordText(); }
  function syncToggles() {
    $('#bMusic').classList.toggle('off', !save.settings.music); $('#bSfx').classList.toggle('off', !save.settings.sfx); $('#vol').value = save.settings.vol;
    document.querySelectorAll('#optOpp button').forEach(b => b.classList.toggle('on', +b.dataset.v === save.settings.opponents));
    document.querySelectorAll('#optSpeed button').forEach(b => b.classList.toggle('on', b.dataset.v === save.settings.speed));
  }
  function startAudio() { Sound.unlock(); Sound.setVolume(save.settings.vol / 100); Sound.setSfx(save.settings.sfx); if (save.settings.music) Sound.startMusic(); }
  document.querySelectorAll('#optOpp button').forEach(b => b.onclick = () => { save.settings.opponents = +b.dataset.v; syncToggles(); persist(); Sound.play('tick'); });
  document.querySelectorAll('#optSpeed button').forEach(b => b.onclick = () => { save.settings.speed = b.dataset.v; syncToggles(); persist(); Sound.play('tick'); });
  $('#bStart').onclick = () => { $('#title').classList.add('hidden'); startAudio(); newGame(); };
  $('#bAgain').onclick = () => { Sound.play('tick'); newGame(); };
  $('#bToMenu').onclick = $('#bMenu').onclick = () => { $('#result').classList.add('hidden'); $('#title').classList.remove('hidden'); syncToggles(); showStats(); };
  $('#bNew').onclick = () => { $('#title').classList.add('hidden'); startAudio(); newGame(); };
  $('#bRules').onclick = $('#bHow').onclick = () => $('#rules').classList.remove('hidden');
  $('#bCredits').onclick = () => { $('#credits').classList.remove('hidden'); loadChangelog(); };
  document.querySelectorAll('[data-close]').forEach(b => b.onclick = () => b.closest('.veil').classList.add('hidden'));
  $('#bMusic').onclick = () => { save.settings.music = !save.settings.music; Sound.setMusic(save.settings.music); syncToggles(); persist(); };
  $('#bSfx').onclick = () => { save.settings.sfx = !save.settings.sfx; Sound.setSfx(save.settings.sfx); syncToggles(); persist(); Sound.play('tick'); };
  $('#vol').oninput = e => { save.settings.vol = +e.target.value; Sound.setVolume(save.settings.vol / 100); persist(); };
  document.addEventListener('keydown', e => {
    if (e.key === 'F2') { e.preventDefault(); $('#title').classList.add('hidden'); startAudio(); newGame(); }
    else if (e.key === 'm' || e.key === 'M') $('#bMusic').click();
    else if (e.key === 'Escape') document.querySelectorAll('#rules,#credits').forEach(v => v.classList.add('hidden'));
  });
  document.addEventListener('contextmenu', e => e.preventDefault());
  setInterval(() => { $('#now').textContent = save.settings.music && started ? `♫  ${Sound.tuneName()}` : ''; }, 1000);

  async function loadChangelog() {
    let md = '';
    try { md = await (await fetch('CHANGELOG.md', { cache: 'no-store' })).text(); } catch (e) { }
    const esc = s => s.replace(/[&<>]/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;' }[c]));
    let html = '', inList = false;
    for (const line of md.split(/\r?\n/)) {
      let m;
      if ((m = line.match(/^## (.*)/))) { if (inList) { html += '</ul>'; inList = false; } html += `<h4>${esc(m[1].replace(/[\[\]]/g, ''))}</h4>`; }
      else if ((m = line.match(/^### (.*)/))) { if (inList) { html += '</ul>'; inList = false; } html += `<h5>${esc(m[1])}</h5>`; }
      else if ((m = line.match(/^- (.*)/))) { if (!inList) { html += '<ul>'; inList = true; } html += `<li>${esc(m[1])}</li>`; }
    }
    $('#changelog').innerHTML = html + (inList ? '</ul>' : '') || '<p class="muted">No changes recorded.</p>';
  }
  $('#tuneList').textContent = Sound.tunes.map(t => `“${t}”`).join(', ');
  $('#ver').textContent = 'v' + VERSION;
  document.title = `Old Maid ${VERSION}`;

  // title-card fan
  (() => { const f = $('#titleFan'); const show = [DECK.find(c => c.id === 'H13'), DECK.find(c => c.id === 'D12'), DECK.find(c => c.maid), DECK.find(c => c.id === 'S11'), DECK.find(c => c.id === 'C1')];
    show.forEach((c, i) => { const cv = Art.face(c); cv.style.transform = `translateX(-50%) rotate(${(i - 2) * 13}deg)`; cv.style.zIndex = i === 2 ? 9 : i; if (i === 2) cv.style.bottom = '12px'; f.appendChild(cv); }); })();

  window.addEventListener('resize', () => { measure(); layoutAll(); });
  setInterval(() => fetch('api/ping').catch(() => { }), 15000); fetch('api/ping').catch(() => { });
  measure(); syncToggles(); showStats();
  (() => { const d = Art.doily(420), cv = $('#doily'); cv.width = 420; cv.height = 420; cv.getContext('2d').drawImage(d, 0, 0); })();

  // test hooks (used by the automated checks only)
  window.__om = { get players() { return players; }, get finished() { return finished; }, newGame, onCardClick, get busy() { return !!busyClick; }, get stats() { return save.stats; } };
  if (/[?&]autostart/.test(location.search)) $('#bStart').click();
})();
