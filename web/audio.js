'use strict';
/* =====================================================================
   OLD MAID - music and sound effects, synthesized live (all original).
   Tunes: "Tea at Four" (music-box waltz), "The Parlour Rag" (piano),
          "Lamplight Nocturne" (music box), "Sunday Promenade" (piano).
   ===================================================================== */
const Sound = (() => {
  let ctx = null, master, music, sfx, verb, musicOn = true, sfxOn = true, volume = 0.6;
  let noiseBuf = null, playing = false, tuneIdx = 0, timer = null, nextTime = 0, queue = [], qi = 0;

  function ensure() {
    if (!ctx) {
      const AC = window.AudioContext || window.webkitAudioContext; if (!AC) return false;
      ctx = new AC();
      master = ctx.createGain(); master.gain.value = volume; master.connect(ctx.destination);
      // small parlour room reverb
      verb = ctx.createConvolver(); const len = ctx.sampleRate * 1.6, ir = ctx.createBuffer(2, len, ctx.sampleRate);
      for (let ch = 0; ch < 2; ch++) { const d = ir.getChannelData(ch); for (let i = 0; i < len; i++) d[i] = (Math.random() * 2 - 1) * Math.pow(1 - i / len, 3.2); }
      verb.buffer = ir; const wet = ctx.createGain(); wet.gain.value = 0.22; verb.connect(wet); wet.connect(master);
      music = ctx.createGain(); music.gain.value = musicOn ? 0.55 : 0; music.connect(master); music.connect(verb);
      sfx = ctx.createGain(); sfx.gain.value = sfxOn ? 0.9 : 0; sfx.connect(master); sfx.connect(verb);
      noiseBuf = ctx.createBuffer(1, ctx.sampleRate, ctx.sampleRate); const nd = noiseBuf.getChannelData(0); for (let i = 0; i < nd.length; i++) nd[i] = Math.random() * 2 - 1;
    }
    if (ctx.state === 'suspended') ctx.resume();
    return true;
  }
  const hz = m => 440 * Math.pow(2, (m - 69) / 12);

  /* ---------- instruments ---------- */
  function musicBox(t, m, vel = 0.5, out = music) {
    const f = hz(m), g = ctx.createGain(); g.connect(out);
    g.gain.setValueAtTime(0, t); g.gain.linearRampToValueAtTime(vel * 0.32, t + 0.004); g.gain.exponentialRampToValueAtTime(0.0008, t + 2.2);
    for (const [mult, amp] of [[1, 1], [3.01, 0.22], [5.43, 0.08], [2, 0.12]]) {
      const o = ctx.createOscillator(), og = ctx.createGain(); o.type = 'sine'; o.frequency.value = f * mult; og.gain.value = amp;
      if (mult > 2) { og.gain.setValueAtTime(amp, t); og.gain.exponentialRampToValueAtTime(0.001, t + 0.4); }
      o.connect(og); og.connect(g); o.start(t); o.stop(t + 2.3);
    }
  }
  function piano(t, m, dur, vel = 0.5, out = music) {
    const f = hz(m), g = ctx.createGain(), lp = ctx.createBiquadFilter();
    lp.type = 'lowpass'; lp.frequency.setValueAtTime(Math.min(9000, f * 9), t); lp.frequency.exponentialRampToValueAtTime(Math.max(300, f * 2), t + 1.2); lp.Q.value = 0.4;
    g.gain.setValueAtTime(0, t); g.gain.linearRampToValueAtTime(vel * 0.28, t + 0.005); g.gain.exponentialRampToValueAtTime(vel * 0.1, t + 0.25);
    const end = t + Math.max(0.25, dur) + 0.35; g.gain.setValueAtTime(vel * 0.1, end - 0.35); g.gain.exponentialRampToValueAtTime(0.0008, end);
    lp.connect(g); g.connect(out);
    for (const [type, mult, amp, det] of [['triangle', 1, 0.8, 0], ['sawtooth', 1, 0.14, 3], ['sine', 2, 0.18, -2]]) {
      const o = ctx.createOscillator(), og = ctx.createGain(); o.type = type; o.frequency.value = f * mult; o.detune.value = det; og.gain.value = amp;
      o.connect(og); og.connect(lp); o.start(t); o.stop(end + 0.05);
    }
  }
  function noise(t, dur, freq, q, vel, out = sfx, type = 'bandpass') {
    const src = ctx.createBufferSource(); src.buffer = noiseBuf;
    const f = ctx.createBiquadFilter(); f.type = type; f.frequency.value = freq; f.Q.value = q;
    const g = ctx.createGain(); g.gain.setValueAtTime(0, t); g.gain.linearRampToValueAtTime(vel, t + 0.006); g.gain.exponentialRampToValueAtTime(0.001, t + dur);
    src.connect(f); f.connect(g); g.connect(out); src.start(t, Math.random() * 0.5); src.stop(t + dur + 0.02);
  }

  /* ---------- tunes ---------- */
  const NOTE = { C: 0, D: 2, E: 4, F: 5, G: 7, A: 9, B: 11 };
  const midi = n => { const m = n.match(/^([A-G])([#b]?)(-?\d)$/); return 12 * (+m[3] + 1) + NOTE[m[1]] + (m[2] === '#' ? 1 : m[2] === 'b' ? -1 : 0); };
  const parse = str => str.trim().split(/\s+/).filter(x => x !== '|').map(t => { const [n, d] = t.split(':'); return [n === 'r' ? null : midi(n), +d]; });
  function chord(name) {
    const m = name.match(/^([A-G])([#b]?)(m?)(7?)$/); const pc = (NOTE[m[1]] + (m[2] === '#' ? 1 : m[2] === 'b' ? -1 : 0) + 12) % 12;
    const iv = m[3] ? [0, 3, 7] : [0, 4, 7]; if (m[4]) iv.push(10);
    return { bass: 36 + ((pc - 36) % 12 + 12) % 12, tones: iv.map(i => 53 + ((pc + i - 53) % 12 + 12) % 12).sort((a, b) => a - b) };
  }
  const TUNES = [
    { name: 'Tea at Four', bpm: 126, beats: 3, inst: 'box', style: 'waltz',
      chords: 'F F C7 C7 C7 C7 F F F F Bb Bb F C7 F F',
      mel: `A4:2 C5:1 | F5:2 E5:1 | D5:1 C5:1 Bb4:1 | G4:3 | G4:2 Bb4:1 | E5:2 D5:1 | C5:1 A4:1 F4:1 | C5:3 |
            A4:2 C5:1 | F5:2 A5:1 | G5:1 F5:1 D5:1 | Bb4:3 | A4:1 C5:1 F5:1 | E5:2 G5:1 | F5:1 C5:1 A4:1 | F4:3` },
    { name: 'The Parlour Rag', bpm: 92, beats: 2, inst: 'piano', style: 'stride',
      chords: 'C C G7 G7 C C D7 G7 C C7 F Fm C G7 C C',
      mel: `E5:.5 G5:.5 E5:.25 C5:.25 E5:.5 | G5:.75 A5:.25 G5:1 | F5:.5 D5:.5 B4:.5 D5:.5 | F5:.75 E5:.25 D5:1 |
            E5:.5 G5:.5 C6:.5 B5:.5 | A5:.5 G5:.5 E5:1 | F#5:.5 A5:.5 F#5:.5 D5:.5 | G5:.75 F5:.25 D5:.5 B4:.5 |
            C5:.5 E5:.5 G5:.5 C6:.5 | Bb5:.75 A5:.25 G5:1 | A5:.5 F5:.5 C5:.5 F5:.5 | Ab5:.75 G5:.25 F5:1 |
            E5:.5 G5:.5 E5:.5 C5:.5 | D5:.5 F5:.5 B4:.5 D5:.5 | C5:.5 E5:.25 G5:.25 C6:.5 G5:.5 | C6:1.5 r:.5` },
    { name: 'Lamplight Nocturne', bpm: 84, beats: 3, inst: 'box', style: 'arp',
      chords: 'Em Em Am Am D D G B7 Em C Am B7 Em Am B7 Em',
      mel: `B4:2 E5:1 | G5:2 F#5:1 | E5:1 C5:1 A4:1 | B4:3 | A4:2 D5:1 | F#5:2 E5:1 | D5:1 B4:1 G4:1 | F#4:3 |
            E4:1 G4:1 B4:1 | E5:2 D5:1 | C5:1 A4:1 E5:1 | D#5:3 | E5:2 G5:1 | F#5:1 E5:1 C5:1 | B4:2 D#5:1 | E5:3` },
    { name: 'Sunday Promenade', bpm: 108, beats: 4, inst: 'piano', style: 'march',
      chords: 'G G C G D7 D7 G D7 G G C A7 G D7 G G',
      mel: `D5:1 G5:1 F#5:.5 E5:.5 D5:1 | B4:1.5 C5:.5 D5:2 | E5:1 G5:1 E5:1 C5:1 | B4:2 G4:2 |
            A4:1 D5:1 C5:.5 B4:.5 A4:1 | F#4:1.5 G4:.5 A4:2 | B4:1 D5:1 G5:1 F#5:1 | A5:3 r:1 |
            D5:1 G5:1 F#5:.5 E5:.5 D5:1 | B5:1.5 A5:.5 G5:2 | E5:1 G5:1 C6:1 E5:1 | C#5:2 A4:2 |
            B4:1 D5:1 G5:1 B4:1 | A4:1 C5:1 F#5:1 A4:1 | G4:1 B4:.5 D5:.5 G5:1 D5:1 | G5:3 r:1` }
  ];

  function buildQueue(tune) {
    const spb = 60 / tune.bpm, ev = [], chords = tune.chords.split(' ');
    let t = 0;
    for (const [m, d] of parse(tune.mel)) { if (m !== null) ev.push({ t, m, d: d * spb, v: 0.62, kind: 'mel' }); t += d * spb; }
    const bar = tune.beats * spb;
    chords.forEach((cn, i) => {
      const c = chord(cn), b0 = i * bar;
      if (tune.style === 'waltz') {
        ev.push({ t: b0, m: c.bass, d: spb, v: 0.42 }); for (const k of [1, 2]) for (const n of c.tones) ev.push({ t: b0 + k * spb, m: n, d: spb * 0.8, v: 0.16 });
      } else if (tune.style === 'arp') {
        ev.push({ t: b0, m: c.bass + 12, d: bar, v: 0.3 }); c.tones.forEach((n, j) => ev.push({ t: b0 + (j + 1) * spb * 0.66, m: n, d: spb, v: 0.14 }));
      } else if (tune.style === 'stride') {
        for (let k = 0; k < tune.beats; k++) { ev.push({ t: b0 + k * spb, m: k % 2 ? c.bass + 7 : c.bass, d: spb * 0.45, v: 0.4 }); for (const n of c.tones) ev.push({ t: b0 + k * spb + spb / 2, m: n, d: spb * 0.4, v: 0.15 }); }
      } else {
        for (let k = 0; k < tune.beats; k++) { ev.push({ t: b0 + k * spb, m: k % 2 ? c.bass + 7 : c.bass, d: spb * 0.5, v: 0.36 }); if (k % 2) for (const n of c.tones) ev.push({ t: b0 + k * spb, m: n, d: spb * 0.5, v: 0.13 }); }
      }
    });
    ev.sort((a, b) => a.t - b.t);
    return { ev, len: chords.length * bar };
  }

  let cur = null, startAt = 0;
  function schedule() {
    if (!playing || !ctx) return;
    const ahead = ctx.currentTime + 0.3;
    while (cur && qi < cur.ev.length && startAt + cur.ev[qi].t < ahead) {
      const e = cur.ev[qi++], t = Math.max(ctx.currentTime, startAt + e.t);
      if (TUNES[tuneIdx].inst === 'box') musicBox(t, e.m, e.v * 1.25); else piano(t, e.m, e.d, e.v);
    }
    if (cur && qi >= cur.ev.length && ctx.currentTime > startAt + cur.len - 0.2) {   // next tune after a breath
      tuneIdx = (tuneIdx + 1) % TUNES.length; startTune(startAt + cur.len + 1.6);
    }
  }
  function startTune(at) { cur = buildQueue(TUNES[tuneIdx]); qi = 0; startAt = at; }

  /* ---------- sound effects ---------- */
  const fx = {
    slide() { const t = ctx.currentTime; noise(t, 0.16, 2600, 0.7, 0.32); noise(t + 0.02, 0.08, 5200, 1.2, 0.12); },
    flip() { const t = ctx.currentTime; noise(t, 0.05, 3800, 1.5, 0.35, sfx, 'highpass'); noise(t + 0.03, 0.04, 1800, 2, 0.15); },
    shuffle() { const t = ctx.currentTime; for (let i = 0; i < 18; i++) noise(t + i * 0.045 + Math.random() * 0.01, 0.06, 2200 + Math.random() * 1600, 1, 0.14); },
    deal() { noise(ctx.currentTime, 0.1, 3000, 0.9, 0.2); },
    pair() { const t = ctx.currentTime; musicBox(t, 88, 0.6, sfx); musicBox(t + 0.09, 95, 0.5, sfx); },
    out() { const t = ctx.currentTime; [79, 83, 86, 91].forEach((m, i) => musicBox(t + i * 0.08, m, 0.55, sfx)); },
    tick() { noise(ctx.currentTime, 0.03, 4200, 2, 0.15, sfx, 'highpass'); },
    hover() { noise(ctx.currentTime, 0.04, 3200, 3, 0.05); },
    maid() {   // a dramatic little "gasp" chord for drawing the Old Maid
      const t = ctx.currentTime;
      for (const m of [43, 49, 55, 58]) piano(t, m, 0.9, 0.42, sfx);
      for (const m of [61, 67]) piano(t + 0.28, m, 0.8, 0.3, sfx);
    },
    phew() { const t = ctx.currentTime; [72, 76, 79].forEach((m, i) => piano(t + i * 0.07, m, 0.3, 0.34, sfx)); },
    win() {
      const t = ctx.currentTime;
      [60, 64, 67, 72, 76, 79, 84].forEach((m, i) => { piano(t + i * 0.09, m, 0.5, 0.45, sfx); musicBox(t + i * 0.09, m + 12, 0.35, sfx); });
      for (const m of [60, 64, 67, 72]) piano(t + 0.75, m, 1.4, 0.32, sfx);
    },
    lose() {   // a sad trombone, Victorian edition
      const t = ctx.currentTime, notes = [58, 57, 56, 55];
      notes.forEach((m, i) => {
        const st = t + i * 0.42, d = i === 3 ? 1.1 : 0.36;
        const o = ctx.createOscillator(), lp = ctx.createBiquadFilter(), g = ctx.createGain();
        o.type = 'sawtooth'; o.frequency.setValueAtTime(hz(m - 12), st);
        if (i === 3) { const lfo = ctx.createOscillator(), lg = ctx.createGain(); lfo.frequency.value = 6; lg.gain.value = 6; lfo.connect(lg); lg.connect(o.frequency); lfo.start(st); lfo.stop(st + d); }
        lp.type = 'lowpass'; lp.frequency.setValueAtTime(400, st); lp.frequency.linearRampToValueAtTime(1100, st + 0.08); lp.frequency.linearRampToValueAtTime(600, st + d);
        g.gain.setValueAtTime(0, st); g.gain.linearRampToValueAtTime(0.32, st + 0.04); g.gain.setValueAtTime(0.3, st + d - 0.08); g.gain.linearRampToValueAtTime(0, st + d);
        o.connect(lp); lp.connect(g); g.connect(sfx); o.start(st); o.stop(st + d + 0.05);
      });
    }
  };

  return {
    unlock() { ensure(); },
    play(name) { if (!sfxOn || !ensure()) return; try { fx[name] && fx[name](); } catch (e) { } },
    startMusic() {
      if (!ensure()) return; if (playing) return; playing = true;
      startTune(ctx.currentTime + 0.3); clearInterval(timer); timer = setInterval(schedule, 60);
    },
    setMusic(on) {
      musicOn = on; if (!ctx) { if (on) this.startMusic(); return; }
      music.gain.setTargetAtTime(on ? 0.55 : 0, ctx.currentTime, 0.15);
      if (on) this.startMusic();
    },
    setSfx(on) { sfxOn = on; if (ctx) sfx.gain.value = on ? 0.9 : 0; },
    setVolume(v) { volume = v; if (ctx) master.gain.setTargetAtTime(v, ctx.currentTime, 0.05); },
    tuneName() { return TUNES[tuneIdx].name; },
    tunes: TUNES.map(t => t.name)
  };
})();
