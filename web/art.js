'use strict';
/* =====================================================================
   OLD MAID - A Victorian Parlor Game
   All card art is drawn here in code (original artwork, (c) Ironite).
   ===================================================================== */
const Art = (() => {
  const CW = 360, CH = 504;              // card canvas size (HD; CSS scales it down)
  const INK = '#2b1d14';
  const RED = '#8f1d22', BLACK = '#1e1712', GOLD = '#b8913f', GOLD2 = '#e2c27a';
  const IVORY1 = '#fbf6e8', IVORY2 = '#efe2c2';
  const SERIF = '"IM Fell English", Georgia, "Times New Roman", serif';
  const SC = '"IM Fell English SC", "IM Fell English", Georgia, serif';
  const TAU = Math.PI * 2;

  // Suit palettes for the court figures.
  const PAL = {
    H: { coat: '#9b2430', dark: '#6a141c', accent: '#d9a441', ink: RED },
    D: { coat: '#6b3a78', dark: '#45214f', accent: '#d9a441', ink: RED },
    S: { coat: '#273a5c', dark: '#16223a', accent: '#c9a24c', ink: BLACK },
    C: { coat: '#2f5a3c', dark: '#1b3824', accent: '#c9a24c', ink: BLACK }
  };

  /* ---------- small helpers ---------- */
  function rng(seed) { let s = seed >>> 0 || 1; return () => (s = (s * 1664525 + 1013904223) >>> 0) / 4294967296; }
  function rrect(c, x, y, w, h, r) { c.beginPath(); c.moveTo(x + r, y); c.arcTo(x + w, y, x + w, y + h, r); c.arcTo(x + w, y + h, x, y + h, r); c.arcTo(x, y + h, x, y, r); c.arcTo(x, y, x + w, y, r); c.closePath(); }
  function canvas(w, h) { const c = document.createElement('canvas'); c.width = w; c.height = h; return c; }

  /* ---------- suit shapes (drawn, so they look the same on every PC) ---------- */
  function heartPath(c, x, y, s) {
    c.moveTo(x, y + s * 0.45);
    c.bezierCurveTo(x - s * 0.62, y + s * 0.02, x - s * 0.52, y - s * 0.55, x, y - s * 0.2);
    c.bezierCurveTo(x + s * 0.52, y - s * 0.55, x + s * 0.62, y + s * 0.02, x, y + s * 0.45);
  }
  function suit(c, s, x, y, size, color) {
    c.save(); c.fillStyle = color; c.beginPath();
    if (s === 'H') heartPath(c, x, y, size);
    else if (s === 'D') { c.moveTo(x, y - size * 0.5); c.quadraticCurveTo(x + size * 0.18, y - size * 0.18, x + size * 0.4, y); c.quadraticCurveTo(x + size * 0.18, y + size * 0.18, x, y + size * 0.5); c.quadraticCurveTo(x - size * 0.18, y + size * 0.18, x - size * 0.4, y); c.quadraticCurveTo(x - size * 0.18, y - size * 0.18, x, y - size * 0.5); }
    else if (s === 'S') {
      c.save(); c.translate(x, y - size * 0.08); c.rotate(Math.PI); heartPath(c, 0, 0, size * 0.9); c.restore();
      c.moveTo(x, y + size * 0.05); c.quadraticCurveTo(x - size * 0.05, y + size * 0.38, x - size * 0.24, y + size * 0.5); c.lineTo(x + size * 0.24, y + size * 0.5); c.quadraticCurveTo(x + size * 0.05, y + size * 0.38, x, y + size * 0.05);
    } else {
      const r = size * 0.205;
      c.arc(x, y - size * 0.24, r, 0, TAU); c.moveTo(x - size * 0.22 + r, y + size * 0.06); c.arc(x - size * 0.22, y + size * 0.06, r, 0, TAU);
      c.moveTo(x + size * 0.22 + r, y + size * 0.06); c.arc(x + size * 0.22, y + size * 0.06, r, 0, TAU);
      c.moveTo(x + size * 0.12, y); c.arc(x, y, size * 0.12, 0, TAU);
      c.moveTo(x, y + size * 0.02); c.quadraticCurveTo(x - size * 0.04, y + size * 0.38, x - size * 0.22, y + size * 0.5); c.lineTo(x + size * 0.22, y + size * 0.5); c.quadraticCurveTo(x + size * 0.04, y + size * 0.38, x, y + size * 0.02);
    }
    c.fill(); c.restore();
  }

  /* ---------- paper ---------- */
  function paper(c, w, h, seed, r = 22) {
    const g = c.createLinearGradient(0, 0, w, h); g.addColorStop(0, IVORY1); g.addColorStop(1, IVORY2);
    rrect(c, 1, 1, w - 2, h - 2, r); c.fillStyle = g; c.fill();
    c.save(); c.clip();
    const R = rng(seed);
    for (let i = 0; i < 900; i++) { c.fillStyle = `rgba(120,90,40,${R() * 0.05})`; c.fillRect(R() * w, R() * h, 1 + R() * 2, 1 + R() * 2); }
    const v = c.createRadialGradient(w / 2, h / 2, h * 0.3, w / 2, h / 2, h * 0.75); v.addColorStop(0, 'rgba(0,0,0,0)'); v.addColorStop(1, 'rgba(120,80,30,.16)');
    c.fillStyle = v; c.fillRect(0, 0, w, h); c.restore();
    rrect(c, 1, 1, w - 2, h - 2, r); c.strokeStyle = 'rgba(70,50,30,.45)'; c.lineWidth = 2; c.stroke();
  }
  function border(c, w, h, color) {
    c.save(); c.strokeStyle = color; c.globalAlpha = 0.85;
    c.lineWidth = 2.5; rrect(c, 14, 14, w - 28, h - 28, 14); c.stroke();
    c.lineWidth = 1; rrect(c, 20, 20, w - 40, h - 40, 10); c.stroke();
    // little corner flourishes
    for (const [x, y, sx, sy] of [[20, 20, 1, 1], [w - 20, 20, -1, 1], [20, h - 20, 1, -1], [w - 20, h - 20, -1, -1]]) {
      c.beginPath(); c.moveTo(x + sx * 4, y + sy * 34); c.quadraticCurveTo(x + sx * 4, y + sy * 4, x + sx * 34, y + sy * 4); c.stroke();
    }
    c.restore();
  }
  function index(c, rank, s, color, w, h) {
    const draw = () => {
      c.fillStyle = color; c.textAlign = 'center'; c.textBaseline = 'alphabetic';
      c.font = `${rank === '10' ? 46 : 54}px ${SERIF}`;
      c.fillText(rank, 46, 76);
      suit(c, s, 46, 104, 34, color);
    };
    draw(); c.save(); c.translate(w, h); c.rotate(Math.PI); draw(); c.restore();
  }

  /* ---------- Victorian portrait (shared by court cards, opponents and the Old Maid) ----------
     Drawn around (0,0) = bottom centre of the bust. Head is ~60px wide at scale 1. */
  function bust(c, o) {
    const skin = o.skin || '#f2d4b6', hair = o.hair || '#5a3a22';
    c.save(); c.lineJoin = 'round'; c.lineCap = 'round'; c.strokeStyle = INK; c.lineWidth = 2.6;
    // back hair (buns, ringlets, long hair) ---------------------------------
    c.fillStyle = hair;
    if (o.hairStyle === 'updo' || o.hairStyle === 'ringlets' || o.hairStyle === 'bun') {
      c.beginPath(); c.arc(0, -150, 17, 0, TAU); c.fill(); c.stroke();
    }
    if (o.hairStyle === 'ringlets') {
      for (const sx of [-1, 1]) for (let i = 0; i < 4; i++) { c.beginPath(); c.arc(sx * (27 + (i % 2) * 3), -118 + i * 11, 6.5, 0, TAU); c.fill(); c.stroke(); }
    }
    // shoulders / costume ---------------------------------------------------
    c.beginPath(); c.moveTo(-80, 0); c.bezierCurveTo(-78, -42, -54, -60, -22, -64); c.lineTo(22, -64); c.bezierCurveTo(54, -60, 78, -42, 80, 0); c.closePath();
    c.fillStyle = o.coat; c.fill(); c.stroke();
    if (o.sleeves) for (const sx of [-1, 1]) { c.beginPath(); c.ellipse(sx * 62, -36, 22, 28, sx * 0.3, 0, TAU); c.fillStyle = o.coat; c.fill(); c.stroke(); c.beginPath(); c.moveTo(sx * 50, -56); c.quadraticCurveTo(sx * 64, -40, sx * 52, -14); c.strokeStyle = 'rgba(0,0,0,.25)'; c.stroke(); c.strokeStyle = INK; }
    // neck
    c.fillStyle = skin; c.beginPath(); c.moveTo(-11, -88); c.lineTo(-12, -60); c.lineTo(12, -60); c.lineTo(11, -88); c.closePath(); c.fill();
    if (o.collar === 'cravat') {
      c.fillStyle = '#f7f1e3'; c.beginPath(); c.moveTo(-20, -64); c.lineTo(0, -16); c.lineTo(20, -64); c.closePath(); c.fill(); c.stroke();
      c.fillStyle = o.dark; for (const sx of [-1, 1]) { c.beginPath(); c.moveTo(sx * 22, -64); c.lineTo(sx * 4, -16); c.lineTo(sx * 36, -30); c.closePath(); c.fill(); c.stroke(); }
      c.fillStyle = o.accent; c.beginPath(); c.moveTo(0, -56); c.lineTo(-14, -64); c.lineTo(-14, -48); c.closePath(); c.moveTo(0, -56); c.lineTo(14, -64); c.lineTo(14, -48); c.closePath(); c.fill(); c.stroke();
      c.beginPath(); c.arc(0, -56, 4, 0, TAU); c.fill(); c.stroke();
      c.fillStyle = o.accent; for (const y of [-34, -22]) { c.beginPath(); c.arc(0, y, 2.6, 0, TAU); c.fill(); }
    } else if (o.collar === 'lace') {
      c.fillStyle = '#fbf7ee';
      c.beginPath(); c.moveTo(-24, -60); c.lineTo(-17, -92); c.lineTo(17, -92); c.lineTo(24, -60); c.closePath(); c.fill(); c.stroke();
      for (let i = -3; i <= 3; i++) { c.beginPath(); c.arc(i * 7, -92, 3.6, Math.PI, 0); c.fill(); c.stroke(); }
      c.strokeStyle = 'rgba(80,60,40,.35)'; c.lineWidth = 1; for (let y = -86; y < -62; y += 6) { c.beginPath(); c.moveTo(-18, y); c.lineTo(18, y); c.stroke(); }
      c.strokeStyle = INK; c.lineWidth = 2.6;
      c.fillStyle = o.accent; c.beginPath(); c.ellipse(0, -56, 9, 11, 0, 0, TAU); c.fill(); c.stroke();
      c.fillStyle = '#f3e6d6'; c.beginPath(); c.ellipse(0, -56, 5, 7, 0, 0, TAU); c.fill();
      if (o.pearls) { c.fillStyle = '#fffaf0'; for (let i = 0; i < 9; i++) { const a = Math.PI * (0.15 + i * 0.0875); c.beginPath(); c.arc(Math.cos(a) * 34, -64 + Math.sin(a) * 26, 3.2, 0, TAU); c.fill(); c.stroke(); } }
    } else if (o.collar === 'buttons') {
      c.fillStyle = o.dark; c.beginPath(); c.moveTo(-20, -64); c.lineTo(20, -64); c.lineTo(18, -54); c.lineTo(-18, -54); c.closePath(); c.fill(); c.stroke();
      c.strokeStyle = o.accent; c.lineWidth = 2; c.beginPath(); c.moveTo(-18, -58); c.lineTo(18, -58); c.stroke(); c.strokeStyle = INK; c.lineWidth = 2.6;
      c.fillStyle = o.accent; for (const y of [-42, -28, -14]) for (const sx of [-1, 1]) { c.beginPath(); c.arc(sx * 11, y, 3.6, 0, TAU); c.fill(); c.stroke(); }
      c.strokeStyle = o.accent; c.lineWidth = 2; for (const sx of [-1, 1]) { c.beginPath(); c.moveTo(sx * 40, -54); c.lineTo(sx * 58, -50); c.stroke(); } c.strokeStyle = INK; c.lineWidth = 2.6;
    } else if (o.collar === 'military') {
      c.fillStyle = o.dark; c.beginPath(); c.moveTo(-18, -64); c.lineTo(18, -64); c.lineTo(16, -52); c.lineTo(-16, -52); c.closePath(); c.fill(); c.stroke();
      c.fillStyle = o.accent; for (const sx of [-1, 1]) { c.beginPath(); rrect(c, sx * 58 - 16, -60, 32, 10, 4); c.fill(); c.stroke(); for (let i = 0; i < 4; i++) { c.beginPath(); c.moveTo(sx * 58 - 14 + i * 9, -50); c.lineTo(sx * 58 - 14 + i * 9, -42); c.strokeStyle = o.accent; c.stroke(); } c.strokeStyle = INK; }
      c.fillStyle = o.accent; for (const y of [-40, -26, -12]) { c.beginPath(); c.arc(0, y, 3.4, 0, TAU); c.fill(); c.stroke(); }
      c.fillStyle = '#d9b44a'; c.beginPath(); c.arc(-34, -30, 7, 0, TAU); c.fill(); c.stroke(); c.fillStyle = '#b02a2a'; c.fillRect(-38, -44, 8, 8);
    } else if (o.collar === 'shawl') {
      c.fillStyle = o.shawl; c.beginPath(); c.moveTo(-82, -6); c.bezierCurveTo(-74, -50, -40, -66, -18, -66); c.lineTo(0, -20); c.lineTo(18, -66); c.bezierCurveTo(40, -66, 74, -50, 82, -6); c.lineTo(40, 0); c.lineTo(0, -10); c.lineTo(-40, 0); c.closePath(); c.fill(); c.stroke();
      c.strokeStyle = 'rgba(60,40,60,.4)'; c.lineWidth = 1.2; for (let i = 0; i < 14; i++) { const x = -78 + i * 12; c.beginPath(); c.moveTo(x, -4 + Math.abs(x) * 0.02); c.lineTo(x - 2, 6); c.stroke(); }
      c.strokeStyle = INK; c.lineWidth = 2.6;
      c.fillStyle = '#fbf7ee'; c.beginPath(); c.moveTo(-14, -62); c.lineTo(-12, -86); c.lineTo(12, -86); c.lineTo(14, -62); c.closePath(); c.fill(); c.stroke();
      c.fillStyle = o.accent; c.beginPath(); c.ellipse(0, -60, 7, 8, 0, 0, TAU); c.fill(); c.stroke();
    }
    // head ---------------------------------------------------------------------
    const hy = -118;
    c.fillStyle = skin;
    c.beginPath(); c.ellipse(-27, hy + 2, 5, 8, 0, 0, TAU); c.ellipse(27, hy + 2, 5, 8, 0, 0, TAU); c.fill(); c.stroke();
    c.beginPath(); c.ellipse(0, hy, 27, 33, 0, 0, TAU); c.fill(); c.stroke();
    // front hair
    c.fillStyle = hair;
    if (o.hairStyle === 'short') { c.beginPath(); c.ellipse(0, hy - 14, 28, 22, 0, Math.PI, 0); c.quadraticCurveTo(14, hy - 22, -4, hy - 18); c.quadraticCurveTo(-18, hy - 14, -28, hy - 6); c.closePath(); c.fill(); c.stroke(); }
    else if (o.hairStyle === 'updo' || o.hairStyle === 'ringlets' || o.hairStyle === 'bun') {
      c.beginPath(); c.moveTo(-29, hy + 6); c.bezierCurveTo(-34, hy - 30, -14, hy - 40, 0, hy - 36); c.bezierCurveTo(14, hy - 40, 34, hy - 30, 29, hy + 6);
      c.bezierCurveTo(22, hy - 10, 14, hy - 24, 0, hy - 22); c.bezierCurveTo(-14, hy - 24, -22, hy - 10, -29, hy + 6); c.closePath(); c.fill(); c.stroke();
      c.strokeStyle = 'rgba(255,255,255,.18)'; c.lineWidth = 1.2; c.beginPath(); c.moveTo(-18, hy - 26); c.quadraticCurveTo(-6, hy - 34, 6, hy - 30); c.stroke(); c.strokeStyle = INK; c.lineWidth = 2.6;
    }
    // sideburns / beard
    if (o.muttonchops) { for (const sx of [-1, 1]) { c.beginPath(); c.moveTo(sx * 26, hy - 10); c.quadraticCurveTo(sx * 30, hy + 18, sx * 14, hy + 24); c.quadraticCurveTo(sx * 18, hy + 6, sx * 22, hy - 8); c.closePath(); c.fill(); c.stroke(); } }
    if (o.beard) { c.beginPath(); c.moveTo(-24, hy + 8); c.quadraticCurveTo(-22, hy + 46, 0, hy + 50); c.quadraticCurveTo(22, hy + 46, 24, hy + 8); c.quadraticCurveTo(12, hy + 24, 0, hy + 22); c.quadraticCurveTo(-12, hy + 24, -24, hy + 8); c.fill(); c.stroke(); }
    // face
    c.fillStyle = 'rgba(214,110,100,.28)'; c.beginPath(); c.arc(-15, hy + 9, 6.5, 0, TAU); c.arc(15, hy + 9, 6.5, 0, TAU); c.fill();
    c.fillStyle = INK;
    c.beginPath(); c.ellipse(-10, hy - 3, 2.6, o.lady ? 3.4 : 3, 0, 0, TAU); c.ellipse(10, hy - 3, 2.6, o.lady ? 3.4 : 3, 0, 0, TAU); c.fill();
    c.lineWidth = 2; c.strokeStyle = o.brow || hair;
    c.beginPath(); c.moveTo(-15, hy - 11 + (o.stern ? 2 : 0)); c.lineTo(-5, hy - 12 - (o.stern ? 0 : 1)); c.moveTo(15, hy - 11 + (o.stern ? 2 : 0)); c.lineTo(5, hy - 12 - (o.stern ? 0 : 1)); c.stroke();
    c.strokeStyle = INK; c.lineWidth = 1.8;
    c.beginPath(); c.moveTo(1, hy); c.quadraticCurveTo(-3, hy + 9, 1, hy + 11); c.stroke();
    if (o.lady) { c.fillStyle = '#b4423f'; c.beginPath(); c.ellipse(0, hy + 20, o.stern ? 5 : 6, o.stern ? 1.8 : 2.8, 0, 0, TAU); c.fill(); }
    else if (!o.mustache) { c.beginPath(); c.moveTo(-6, hy + 19); c.quadraticCurveTo(0, hy + 22, 6, hy + 19); c.stroke(); }
    if (o.mustache) {
      c.fillStyle = o.mustacheColor || hair; c.strokeStyle = INK; c.lineWidth = 1.6;
      c.beginPath(); c.moveTo(0, hy + 13); c.bezierCurveTo(-8, hy + 10, -18, hy + 14, -26, hy + 8); c.bezierCurveTo(-22, hy + 20, -8, hy + 20, 0, hy + 17);
      c.bezierCurveTo(8, hy + 20, 22, hy + 20, 26, hy + 8); c.bezierCurveTo(18, hy + 14, 8, hy + 10, 0, hy + 13); c.fill(); c.stroke();
    }
    c.lineWidth = 2.6; c.strokeStyle = INK;
    if (o.monocle) { c.strokeStyle = GOLD; c.lineWidth = 2.2; c.beginPath(); c.arc(10, hy - 3, 8, 0, TAU); c.stroke(); c.beginPath(); c.moveTo(17, hy + 1); c.quadraticCurveTo(30, hy + 30, 22, hy + 52); c.stroke(); c.strokeStyle = INK; c.lineWidth = 2.6; }
    if (o.spectacles) { c.strokeStyle = '#6d5a48'; c.lineWidth = 2; c.beginPath(); c.arc(-10, hy - 3, 7.5, 0, TAU); c.moveTo(-2.5, hy - 3); c.arc(10, hy - 3, 7.5, Math.PI, Math.PI * 3); c.moveTo(-2.5, hy - 4); c.lineTo(2.5, hy - 4); c.moveTo(-17.5, hy - 4); c.lineTo(-26, hy - 7); c.moveTo(17.5, hy - 4); c.lineTo(26, hy - 7); c.stroke(); c.strokeStyle = INK; c.lineWidth = 2.6; }
    // hats -------------------------------------------------------------------
    if (o.hat === 'top') {
      c.fillStyle = '#1d1712'; c.beginPath(); c.moveTo(-23, hy - 30); c.lineTo(-25, hy - 86); c.quadraticCurveTo(0, hy - 92, 25, hy - 86); c.lineTo(23, hy - 30); c.closePath(); c.fill(); c.stroke();
      c.fillStyle = o.accent2 || o.coat; c.fillRect(-23, hy - 42, 46, 9); c.strokeRect(-23, hy - 42, 46, 9);
      c.fillStyle = '#1d1712'; c.beginPath(); c.ellipse(0, hy - 30, 40, 7, 0, 0, TAU); c.fill(); c.stroke();
      c.strokeStyle = 'rgba(255,255,255,.15)'; c.lineWidth = 3; c.beginPath(); c.moveTo(-14, hy - 82); c.lineTo(-13, hy - 46); c.stroke();
    } else if (o.hat === 'feather') {
      c.save(); c.translate(10, hy - 36); c.rotate(0.22);
      c.fillStyle = o.hatColor || o.dark; c.beginPath(); c.ellipse(0, 4, 30, 7, 0, 0, TAU); c.fill(); c.stroke();
      c.beginPath(); c.ellipse(0, -4, 17, 12, 0, Math.PI, 0); c.closePath(); c.fill(); c.stroke();
      c.fillStyle = o.accent; c.fillRect(-17, -4, 34, 5);
      c.fillStyle = '#f6efe2'; c.beginPath(); c.moveTo(10, -6); c.bezierCurveTo(30, -36, 54, -34, 52, -16); c.bezierCurveTo(40, -24, 28, -14, 10, -6); c.fill(); c.stroke();
      c.strokeStyle = 'rgba(120,100,80,.6)'; c.lineWidth = 1; for (let i = 0; i < 6; i++) { c.beginPath(); c.moveTo(16 + i * 6, -10 - i * 2.5); c.lineTo(20 + i * 6, -20 - i * 2); c.stroke(); }
      c.restore();
    } else if (o.hat === 'bonnet') {
      c.fillStyle = o.hatColor || '#e9d9b8'; c.beginPath(); c.moveTo(-34, hy + 8); c.bezierCurveTo(-42, hy - 46, 42, hy - 46, 34, hy + 8); c.bezierCurveTo(26, hy - 26, -26, hy - 26, -34, hy + 8); c.fill(); c.stroke();
      c.strokeStyle = o.accent; c.lineWidth = 3; c.beginPath(); c.moveTo(-30, hy + 4); c.quadraticCurveTo(-22, hy + 30, -6, hy + 36); c.moveTo(30, hy + 4); c.quadraticCurveTo(22, hy + 30, 6, hy + 36); c.stroke();
      c.fillStyle = o.accent; c.beginPath(); c.arc(0, hy + 36, 5, 0, TAU); c.fill(); c.strokeStyle = INK; c.lineWidth = 2.6;
      c.fillStyle = '#e46b7a'; for (const [x, y] of [[-26, hy - 22], [-20, hy - 30]]) { c.beginPath(); c.arc(x, y, 5, 0, TAU); c.fill(); c.stroke(); }
    } else if (o.hat === 'pillbox') {
      c.fillStyle = o.coat; c.beginPath(); rrect(c, -18, hy - 52, 36, 20, 4); c.fill(); c.stroke();
      c.strokeStyle = o.accent; c.lineWidth = 2.5; c.beginPath(); c.moveTo(-18, hy - 36); c.lineTo(18, hy - 36); c.moveTo(-18, hy - 48); c.lineTo(18, hy - 48); c.stroke();
      c.beginPath(); c.moveTo(-26, hy - 4); c.quadraticCurveTo(-20, hy + 30, 0, hy + 32); c.stroke(); c.strokeStyle = INK; c.lineWidth = 2.6;
      c.fillStyle = o.accent; c.beginPath(); c.arc(0, hy - 42, 3, 0, TAU); c.fill();
    } else if (o.hat === 'military') {
      c.fillStyle = o.coat; c.beginPath(); c.moveTo(-26, hy - 28); c.lineTo(-28, hy - 52); c.quadraticCurveTo(0, hy - 62, 28, hy - 52); c.lineTo(26, hy - 28); c.closePath(); c.fill(); c.stroke();
      c.fillStyle = '#1d1712'; c.beginPath(); c.moveTo(-28, hy - 28); c.quadraticCurveTo(0, hy - 14, 30, hy - 26); c.lineTo(26, hy - 32); c.lineTo(-26, hy - 32); c.closePath(); c.fill(); c.stroke();
      c.strokeStyle = o.accent; c.lineWidth = 3; c.beginPath(); c.moveTo(-26, hy - 36); c.lineTo(26, hy - 36); c.stroke(); c.strokeStyle = INK; c.lineWidth = 2.6;
      c.fillStyle = o.accent; c.beginPath(); c.arc(0, hy - 46, 6, 0, TAU); c.fill(); c.stroke();
    } else if (o.hat === 'lacecap') {
      c.fillStyle = '#fbf8f0'; c.beginPath(); c.ellipse(0, hy - 26, 30, 18, 0, Math.PI, 0); c.closePath(); c.fill(); c.stroke();
      for (let i = 0; i < 11; i++) { const a = Math.PI + (i / 10) * Math.PI; c.beginPath(); c.arc(Math.cos(a) * 31, hy - 26 + Math.sin(a) * 19, 5, 0, TAU); c.fill(); c.stroke(); }
      c.fillStyle = '#c9a7c9'; c.beginPath(); c.moveTo(-24, hy - 30); c.lineTo(-34, hy - 18); c.lineTo(-28, hy - 14); c.closePath(); c.fill(); c.stroke();
      c.beginPath(); c.moveTo(24, hy - 30); c.lineTo(34, hy - 18); c.lineTo(28, hy - 14); c.closePath(); c.fill(); c.stroke();
    }
    c.restore();
  }

  const COURT = {
    K: s => ({ coat: PAL[s].coat, dark: PAL[s].dark, accent: PAL[s].accent, hair: s === 'S' || s === 'C' ? '#4a3322' : '#7a4a26', hairStyle: 'short', collar: 'cravat', hat: 'top', mustache: true, muttonchops: true, monocle: s === 'S' || s === 'H' }),
    Q: s => ({ lady: true, coat: PAL[s].coat, dark: PAL[s].dark, accent: PAL[s].accent, hair: s === 'H' ? '#8a4a24' : s === 'D' ? '#3e2a1c' : '#6b4a2a', hairStyle: 'updo', collar: 'lace', sleeves: true, hat: 'feather', hatColor: PAL[s].dark, pearls: s === 'D' || s === 'H' }),
    J: s => ({ coat: PAL[s].coat, dark: PAL[s].dark, accent: PAL[s].accent, hair: '#a0612e', hairStyle: 'short', collar: 'buttons', hat: 'pillbox' })
  };

  // what each court figure holds
  function courtProp(c, rank, o) {
    c.save(); c.lineJoin = 'round'; c.lineCap = 'round'; c.strokeStyle = INK; c.lineWidth = 2.4;
    const hand = (x, y) => { c.fillStyle = '#f2d4b6'; c.beginPath(); c.ellipse(x, y, 9, 8, 0, 0, TAU); c.fill(); c.stroke(); };
    if (rank === 'K') {          // walking cane with a gold knob
      c.strokeStyle = '#3a2416'; c.lineWidth = 5; c.beginPath(); c.moveTo(60, 4); c.lineTo(66, -98); c.stroke();
      c.fillStyle = GOLD2; c.strokeStyle = INK; c.lineWidth = 2; c.beginPath(); c.arc(66, -103, 7, 0, TAU); c.fill(); c.stroke();
      hand(60, -40);
    } else if (rank === 'Q') {   // folding fan
      c.save(); c.translate(36, -26); c.rotate(-0.35);
      for (let i = 0; i < 9; i++) { const a = -0.95 + i * 0.22; c.fillStyle = i % 2 ? '#f6efe2' : o.accent; c.beginPath(); c.moveTo(0, 0); c.lineTo(Math.sin(a) * 52, -Math.cos(a) * 52); c.lineTo(Math.sin(a + 0.22) * 52, -Math.cos(a + 0.22) * 52); c.closePath(); c.fill(); c.stroke(); }
      c.restore(); hand(36, -24);
    } else {                     // a sealed letter
      c.save(); c.translate(40, -22); c.rotate(-0.22);
      c.fillStyle = '#f7f0dc'; c.fillRect(-24, -30, 42, 28); c.strokeRect(-24, -30, 42, 28);
      c.beginPath(); c.moveTo(-24, -30); c.lineTo(-3, -14); c.lineTo(18, -30); c.stroke();
      c.fillStyle = '#9b2430'; c.beginPath(); c.arc(-3, -14, 4.5, 0, TAU); c.fill();
      c.restore(); hand(30, -26);
    }
    c.restore();
  }

  function courtCard(c, rank, s) {
    const color = PAL[s].ink, w = CW, h = CH;
    const fx = 62, fy = 70, fw = w - 124, fh = h - 140;
    // frame
    c.save(); rrect(c, fx, fy, fw, fh, 10); c.fillStyle = '#f6edd6'; c.fill(); c.clip();
    // damask-like wallpaper behind the figure
    c.fillStyle = s === 'H' || s === 'D' ? 'rgba(143,29,34,.07)' : 'rgba(30,40,60,.07)';
    for (let y = fy - 10; y < fy + fh; y += 26) for (let x = fx - 10 + ((y / 26) % 2) * 13; x < fx + fw; x += 26) { c.beginPath(); c.moveTo(x, y - 7); c.lineTo(x + 5, y); c.lineTo(x, y + 7); c.lineTo(x - 5, y); c.closePath(); c.fill(); }
    // one full portrait per court card
    const fit = { K: 1.62, Q: 1.72, J: 1.78 }[rank], o = COURT[rank](s);
    const glow = c.createRadialGradient(fx + fw / 2, fy + fh * 0.42, 10, fx + fw / 2, fy + fh * 0.42, fh * 0.6);
    glow.addColorStop(0, 'rgba(255,248,225,.75)'); glow.addColorStop(1, 'rgba(255,248,225,0)'); c.fillStyle = glow; c.fillRect(fx, fy, fw, fh);
    c.save(); c.translate(fx + fw / 2, fy + fh + 2); c.scale(fit, fit);
    bust(c, o); courtProp(c, rank, o);
    c.restore();
    c.restore();
    c.strokeStyle = GOLD; c.lineWidth = 3; rrect(c, fx, fy, fw, fh, 10); c.stroke();
    c.lineWidth = 1; rrect(c, fx + 5, fy + 5, fw - 10, fh - 10, 7); c.stroke();
    suit(c, s, fx + 22, fy + 26, 26, color);
    suit(c, s, fx + fw - 22, fy + 26, 26, color);
  }

  const PIPS = {
    2: [[.5, 0], [.5, 1]], 3: [[.5, 0], [.5, .5], [.5, 1]], 4: [[0, 0], [1, 0], [0, 1], [1, 1]],
    5: [[0, 0], [1, 0], [.5, .5], [0, 1], [1, 1]], 6: [[0, 0], [1, 0], [0, .5], [1, .5], [0, 1], [1, 1]],
    7: [[0, 0], [1, 0], [.5, .25], [0, .5], [1, .5], [0, 1], [1, 1]],
    8: [[0, 0], [1, 0], [.5, .25], [0, .5], [1, .5], [.5, .75], [0, 1], [1, 1]],
    9: [[0, 0], [1, 0], [0, 1 / 3], [1, 1 / 3], [.5, .5], [0, 2 / 3], [1, 2 / 3], [0, 1], [1, 1]],
    10: [[0, 0], [1, 0], [.5, 1 / 6], [0, 1 / 3], [1, 1 / 3], [0, 2 / 3], [1, 2 / 3], [.5, 5 / 6], [0, 1], [1, 1]]
  };
  function pipCard(c, n, s) {
    const color = PAL[s].ink, x0 = 112, x1 = CW - 112, y0 = 104, y1 = CH - 104;
    for (const [px, py] of PIPS[n]) {
      const x = x0 + (x1 - x0) * px, y = y0 + (y1 - y0) * py;
      if (py > 0.5) { c.save(); c.translate(x, y); c.rotate(Math.PI); suit(c, s, 0, 0, 58, color); c.restore(); }
      else suit(c, s, x, y, 58, color);
    }
  }
  function aceCard(c, s) {
    const color = PAL[s].ink, cx = CW / 2, cy = CH / 2;
    c.save(); c.strokeStyle = GOLD; c.lineWidth = 2;
    c.beginPath(); c.ellipse(cx, cy, 98, 128, 0, 0, TAU); c.stroke();
    c.lineWidth = 1; c.beginPath(); c.ellipse(cx, cy, 104, 134, 0, 0, TAU); c.stroke();
    // filigree sprays
    for (let i = 0; i < 12; i++) {
      const a = i / 12 * TAU, r1 = 92, r2 = 118;
      c.beginPath(); c.moveTo(cx + Math.cos(a) * r1 * 0.95, cy + Math.sin(a) * r1 * 1.25);
      c.quadraticCurveTo(cx + Math.cos(a + 0.18) * r2, cy + Math.sin(a + 0.18) * r2 * 1.22, cx + Math.cos(a + 0.3) * r1, cy + Math.sin(a + 0.3) * r1 * 1.28); c.stroke();
    }
    c.restore();
    suit(c, s, cx, cy, 130, color);
    c.fillStyle = 'rgba(255,255,255,.18)'; c.beginPath(); c.ellipse(cx - 24, cy - 28, 14, 24, -0.5, 0, TAU); c.fill();
  }

  /* ---------- the Old Maid ---------- */
  function oldMaidFigure(c, w, h) {
    // wallpaper + window glow
    const g = c.createLinearGradient(0, 0, 0, h); g.addColorStop(0, '#e9dcc0'); g.addColorStop(1, '#d6c3a0'); c.fillStyle = g; c.fillRect(0, 0, w, h);
    c.strokeStyle = 'rgba(120,70,90,.14)'; c.lineWidth = 2;
    for (let x = 10; x < w; x += 30) { c.beginPath(); for (let y = 0; y <= h; y += 6) c.lineTo(x + Math.sin(y / 14) * 6, y); c.stroke(); }
    // portrait frame on the wall
    c.fillStyle = '#7a5a2e'; c.fillRect(w * 0.66, 30, 56, 70); c.fillStyle = '#e9e2cc'; c.fillRect(w * 0.66 + 7, 37, 42, 56);
    c.fillStyle = '#7a6a54'; c.beginPath(); c.ellipse(w * 0.66 + 28, 60, 9, 12, 0, 0, TAU); c.fill(); c.beginPath(); c.ellipse(w * 0.66 + 28, 84, 15, 10, 0, Math.PI, 0); c.fill();
    // the lady herself
    c.save(); c.translate(w / 2, h - 36); c.scale(1.18, 1.18);
    bust(c, { lady: true, stern: true, skin: '#f0d2b8', hair: '#c9c4bc', brow: '#8a847a', hairStyle: 'bun', coat: '#3a2a3e', dark: '#24182a', accent: '#b7a3c9', collar: 'shawl', shawl: '#8a7a9a', hat: 'lacecap', spectacles: true });
    // teacup in hand
    c.save(); c.translate(36, -18);
    c.fillStyle = '#f2d4b6'; c.strokeStyle = INK; c.lineWidth = 2.4; c.beginPath(); c.ellipse(-4, 12, 12, 9, 0, 0, TAU); c.fill(); c.stroke();
    c.fillStyle = '#fbf8f0'; c.beginPath(); c.ellipse(0, 4, 26, 5, 0, 0, TAU); c.fill(); c.stroke();
    c.beginPath(); c.moveTo(-16, -16); c.lineTo(16, -16); c.quadraticCurveTo(14, 4, 0, 4); c.quadraticCurveTo(-14, 4, -16, -16); c.closePath(); c.fill(); c.stroke();
    c.beginPath(); c.arc(19, -8, 6, -1.4, 1.4); c.stroke();
    c.strokeStyle = '#7b9ad0'; c.lineWidth = 2; c.beginPath(); c.moveTo(-12, -10); c.lineTo(12, -10); c.stroke();
    c.strokeStyle = 'rgba(140,140,140,.55)'; c.lineWidth = 2; for (const dx of [-5, 4]) { c.beginPath(); c.moveTo(dx, -22); c.bezierCurveTo(dx - 6, -30, dx + 6, -36, dx, -46); c.stroke(); }
    c.restore();
    c.restore();
    // the cat
    c.save(); c.translate(52, h - 34);
    c.fillStyle = '#1f1a1e'; c.strokeStyle = '#0d0a0c'; c.lineWidth = 2;
    c.beginPath(); c.ellipse(0, 6, 30, 18, 0, 0, TAU); c.fill();
    c.beginPath(); c.arc(-22, -10, 14, 0, TAU); c.fill();
    c.beginPath(); c.moveTo(-33, -18); c.lineTo(-30, -34); c.lineTo(-22, -22); c.moveTo(-20, -23); c.lineTo(-12, -34); c.lineTo(-10, -18); c.fill();
    c.beginPath(); c.moveTo(26, 12); c.bezierCurveTo(50, 10, 46, -14, 34, -12); c.lineWidth = 7; c.strokeStyle = '#1f1a1e'; c.stroke();
    c.fillStyle = '#e8d24a'; c.beginPath(); c.ellipse(-27, -11, 3, 2.4, 0, 0, TAU); c.ellipse(-17, -11, 3, 2.4, 0, 0, TAU); c.fill();
    c.fillStyle = '#0d0a0c'; c.fillRect(-27.6, -13, 1.2, 4); c.fillRect(-17.6, -13, 1.2, 4);
    c.restore();
  }
  function oldMaidCard(c) {
    const w = CW, h = CH, fx = 34, fy = 40, fw = w - 68, fh = h - 128;
    c.save(); rrect(c, fx, fy, fw, fh, 12); c.clip();
    c.translate(fx, fy); oldMaidFigure(c, fw, fh); c.restore();
    c.strokeStyle = '#3a2a3e'; c.lineWidth = 4; rrect(c, fx, fy, fw, fh, 12); c.stroke();
    c.strokeStyle = GOLD; c.lineWidth = 1.5; rrect(c, fx + 6, fy + 6, fw - 12, fh - 12, 8); c.stroke();
    // ribbon banner
    const by = h - 62;
    c.fillStyle = '#3a2a3e'; c.beginPath(); c.moveTo(22, by - 24); c.lineTo(44, by - 24); c.lineTo(44, by + 24); c.lineTo(22, by + 24); c.lineTo(32, by); c.closePath(); c.fill();
    c.beginPath(); c.moveTo(w - 22, by - 24); c.lineTo(w - 44, by - 24); c.lineTo(w - 44, by + 24); c.lineTo(w - 22, by + 24); c.lineTo(w - 32, by); c.closePath(); c.fill();
    c.fillStyle = '#5a3f5e'; c.beginPath(); rrect(c, 40, by - 28, w - 80, 56, 6); c.fill();
    c.strokeStyle = GOLD2; c.lineWidth = 1.5; rrect(c, 46, by - 22, w - 92, 44, 4); c.stroke();
    c.fillStyle = '#f6e7c6'; c.textAlign = 'center'; c.textBaseline = 'middle'; c.font = `34px ${SC}`; c.fillText('The Old Maid', w / 2, by + 2);
  }

  /* ---------- card back ---------- */
  function backCard() {
    const cv = canvas(CW, CH), c = cv.getContext('2d');
    rrect(c, 1, 1, CW - 2, CH - 2, 22); c.fillStyle = '#f4ead2'; c.fill();
    rrect(c, 12, 12, CW - 24, CH - 24, 14);
    const g = c.createLinearGradient(0, 0, CW, CH); g.addColorStop(0, '#6e1f2a'); g.addColorStop(1, '#4a121b'); c.fillStyle = g; c.fill();
    c.save(); c.clip();
    c.strokeStyle = 'rgba(226,194,122,.32)'; c.lineWidth = 1.4;
    for (let i = -CH; i < CW + CH; i += 22) { c.beginPath(); c.moveTo(i, 0); c.lineTo(i + CH, CH); c.moveTo(i, CH); c.lineTo(i + CH, 0); c.stroke(); }
    c.fillStyle = 'rgba(226,194,122,.4)';
    for (let y = 11; y < CH; y += 22) for (let x = ((y - 11) / 22) % 2 ? 22 : 11; x < CW; x += 22) { c.beginPath(); c.arc(x, y, 1.8, 0, TAU); c.fill(); }
    c.restore();
    c.strokeStyle = GOLD2; c.lineWidth = 2.5; rrect(c, 20, 20, CW - 40, CH - 40, 10); c.stroke();
    c.lineWidth = 1; rrect(c, 27, 27, CW - 54, CH - 54, 7); c.stroke();
    // medallion with a teapot silhouette
    const cx = CW / 2, cy = CH / 2;
    c.fillStyle = '#f4ead2'; c.beginPath(); c.ellipse(cx, cy, 78, 100, 0, 0, TAU); c.fill();
    c.strokeStyle = GOLD; c.lineWidth = 4; c.stroke();
    c.lineWidth = 1.2; c.beginPath(); c.ellipse(cx, cy, 70, 92, 0, 0, TAU); c.stroke();
    c.fillStyle = '#4a121b';
    c.beginPath(); c.ellipse(cx, cy + 8, 40, 32, 0, 0, TAU); c.fill();
    c.beginPath(); c.ellipse(cx, cy - 26, 22, 8, 0, 0, TAU); c.fill();
    c.beginPath(); c.arc(cx, cy - 36, 6, 0, TAU); c.fill();
    c.beginPath(); c.moveTo(cx + 34, cy + 2); c.quadraticCurveTo(cx + 62, cy - 2, cx + 66, cy - 26); c.lineTo(cx + 58, cy - 26); c.quadraticCurveTo(cx + 54, cy - 8, cx + 34, cy + 14); c.fill();
    c.lineWidth = 7; c.strokeStyle = '#4a121b'; c.beginPath(); c.arc(cx - 42, cy + 4, 16, Math.PI * 0.5, Math.PI * 1.5); c.stroke();
    c.fillRect(cx - 30, cy + 36, 60, 7);
    c.fillStyle = '#4a121b'; c.font = `22px ${SC}`; c.textAlign = 'center'; c.fillText('Old Maid', cx, cy + 74);
    c.font = `16px ${SERIF}`; c.fillText('~ 1890 ~', cx, cy - 62);
    // corner fleurons
    c.fillStyle = GOLD2;
    for (const [x, y] of [[48, 48], [CW - 48, 48], [48, CH - 48], [CW - 48, CH - 48]]) { c.save(); c.translate(x, y); for (let i = 0; i < 4; i++) { c.rotate(Math.PI / 2); c.beginPath(); c.ellipse(0, -8, 3.5, 8, 0, 0, TAU); c.fill(); } c.restore(); }
    rrect(c, 1, 1, CW - 2, CH - 2, 22); c.strokeStyle = 'rgba(70,50,30,.45)'; c.lineWidth = 2; c.stroke();
    return cv;
  }

  /* ---------- public: a card face ---------- */
  const RANK = { 1: 'A', 11: 'J', 12: 'Q', 13: 'K' };
  function face(card) {
    const cv = canvas(CW, CH), c = cv.getContext('2d');
    paper(c, CW, CH, card.r * 7 + 'SHDC'.indexOf(card.s) * 131);
    if (card.maid) { oldMaidCard(c); return cv; }
    const color = PAL[card.s].ink;
    border(c, CW, CH, GOLD);
    const label = RANK[card.r] || String(card.r);
    index(c, label, card.s, color, CW, CH);
    if (card.r === 1) aceCard(c, card.s);
    else if (card.r > 10) courtCard(c, label, card.s);
    else pipCard(c, card.r, card.s);
    return cv;
  }

  /* ---------- opponents' cameo portraits ---------- */
  const PEOPLE = {
    agatha: { lady: true, stern: true, skin: '#f1d0b5', hair: '#9a9aa4', brow: '#6d6d78', hairStyle: 'bun', coat: '#4b2a5c', dark: '#2e1838', accent: '#d9b44a', collar: 'lace', pearls: true, sleeves: true, hat: 'feather', hatColor: '#2e1838' },
    colonel: { skin: '#eec3a2', hair: '#e8e4dc', mustacheColor: '#f2eee6', brow: '#d8d2c6', hairStyle: 'short', coat: '#a3242b', dark: '#6b1218', accent: '#d9b44a', collar: 'military', hat: 'military', mustache: true, muttonchops: true },
    pettigrew: { lady: true, skin: '#f6dbc4', hair: '#b5651d', hairStyle: 'ringlets', coat: '#2f7a7a', dark: '#1d4f50', accent: '#e8b4b8', collar: 'lace', sleeves: true, hat: 'bonnet', hatColor: '#f1e4c4' },
    maid: { lady: true, stern: true, skin: '#f0d2b8', hair: '#c9c4bc', brow: '#8a847a', hairStyle: 'bun', coat: '#3a2a3e', dark: '#24182a', accent: '#b7a3c9', collar: 'shawl', shawl: '#8a7a9a', hat: 'lacecap', spectacles: true }
  };
  function cameo(who, size) {
    const cv = canvas(size, size), c = cv.getContext('2d'), r = size / 2;
    const g = c.createRadialGradient(r, r * 0.8, r * 0.1, r, r, r); g.addColorStop(0, '#f3e6c6'); g.addColorStop(1, '#c9b083');
    c.beginPath(); c.arc(r, r, r - 2, 0, TAU); c.fillStyle = g; c.fill();
    c.save(); c.beginPath(); c.arc(r, r, r - 7, 0, TAU); c.clip();
    c.translate(r, size * 1.02); const k = size / 190; c.scale(k, k); bust(c, PEOPLE[who]); c.restore();
    c.lineWidth = size * 0.05; c.strokeStyle = GOLD; c.beginPath(); c.arc(r, r, r - size * 0.03, 0, TAU); c.stroke();
    c.lineWidth = 1.5; c.strokeStyle = '#7a5a26'; c.beginPath(); c.arc(r, r, r - size * 0.06, 0, TAU); c.stroke();
    return cv;
  }

  /* ---------- the doily under the discard pile ---------- */
  function doily(size) {
    const cv = canvas(size, size), c = cv.getContext('2d'), r = size / 2;
    c.translate(r, r); c.fillStyle = 'rgba(250,245,232,.92)'; c.strokeStyle = 'rgba(160,140,110,.5)'; c.lineWidth = 1;
    c.beginPath(); for (let i = 0; i <= 48; i++) { const a = i / 48 * TAU, rr = r * (0.9 + (i % 2) * 0.06); c.lineTo(Math.cos(a) * rr, Math.sin(a) * rr); } c.fill();
    c.save(); c.globalCompositeOperation = 'destination-out';
    for (let ring = 0; ring < 4; ring++) {
      const rr = r * (0.82 - ring * 0.16), n = 28 - ring * 5, hole = r * (0.045 - ring * 0.006);
      for (let i = 0; i < n; i++) { const a = i / n * TAU + ring * 0.2; c.beginPath(); c.arc(Math.cos(a) * rr, Math.sin(a) * rr, hole, 0, TAU); c.fill(); }
    }
    for (let i = 0; i < 12; i++) { const a = i / 12 * TAU; c.beginPath(); c.ellipse(Math.cos(a) * r * 0.3, Math.sin(a) * r * 0.3, r * 0.05, r * 0.11, a + Math.PI / 2, 0, TAU); c.fill(); }
    c.restore();
    for (let ring = 0; ring < 3; ring++) { c.beginPath(); c.arc(0, 0, r * (0.75 - ring * 0.18), 0, TAU); c.stroke(); }
    return cv;
  }

  /* ---------- icon / covers (used to build the .exe icon and launcher art) ---------- */
  function icon(size) {
    const cv = canvas(size, size), c = cv.getContext('2d'), s = size / 512;
    c.scale(s, s);
    rrect(c, 16, 16, 480, 480, 96); const g = c.createLinearGradient(0, 0, 512, 512); g.addColorStop(0, '#7a2230'); g.addColorStop(1, '#4a121b'); c.fillStyle = g; c.fill();
    c.strokeStyle = GOLD2; c.lineWidth = 10; rrect(c, 34, 34, 444, 444, 80); c.stroke();
    const cm = cameo('maid', 380); c.drawImage(cm, 66, 66);
    return cv;
  }

  return { face, back: backCard, cameo, doily, icon, CW, CH, PEOPLE, suit, SERIF, SC };
})();
