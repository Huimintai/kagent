'use client'
import { useEffect, useRef } from "react";

interface IdenticonProps {
  value: string;       // username or email used to seed the pattern
  size?: number;       // pixel size of the canvas (default 32)
  className?: string;
}

// Simple string hash → deterministic number
function hashStr(s: string): number {
  let h = 0x811c9dc5;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = (h * 0x01000193) >>> 0;
  }
  return h;
}

// Generate a visually-distinct hue from the hash
function hashToHue(h: number): number {
  return h % 360;
}

export function Identicon({ value, size = 32, className }: IdenticonProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    const seed = value.toLowerCase().trim();
    const hash = hashStr(seed);
    const hue = hashToHue(hash);

    // 5×5 grid, mirrored left-right → 15 unique cells determine the pattern
    const GRID = 5;
    const cellSize = size / GRID;

    const colorFg = `hsl(${hue}, 65%, 48%)`;
    const colorBg = `hsl(${hue}, 20%, 94%)`;

    ctx.clearRect(0, 0, size, size);
    ctx.fillStyle = colorBg;
    ctx.fillRect(0, 0, size, size);

    ctx.fillStyle = colorFg;

    // Use different bytes of the hash to fill a 5×3 boolean grid (mirrored)
    for (let row = 0; row < GRID; row++) {
      for (let col = 0; col < 3; col++) {
        const bitIndex = row * 3 + col;
        const on = (hash >> bitIndex) & 1;
        if (on) {
          // left side
          ctx.fillRect(col * cellSize, row * cellSize, cellSize, cellSize);
          // mirrored right side
          if (col < 2) {
            ctx.fillRect((GRID - 1 - col) * cellSize, row * cellSize, cellSize, cellSize);
          }
        }
      }
    }
  }, [value, size]);

  return (
    <canvas
      ref={canvasRef}
      width={size}
      height={size}
      className={className}
      style={{ borderRadius: "4px", imageRendering: "pixelated" }}
    />
  );
}
