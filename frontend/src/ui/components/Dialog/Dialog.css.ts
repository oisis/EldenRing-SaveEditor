import { keyframes, style } from "@vanilla-extract/css";
import { tokens } from "../../tokens/contract.css";

const fadeIn = keyframes({ from: { opacity: 0 }, to: { opacity: 1 } });
const enter = keyframes({
  from: { opacity: 0, transform: "translate(-50%, -48%) scale(0.98)" },
  to: { opacity: 1, transform: "translate(-50%, -50%) scale(1)" },
});

export const overlay = style({
  position: "fixed",
  inset: 0,
  zIndex: 100,
  backgroundColor: tokens.color.overlay,
  animation: `${fadeIn} ${tokens.motion.fast} ease-out`,
});

export const content = style({
  position: "fixed",
  top: "50%",
  left: "50%",
  zIndex: 101,
  display: "flex",
  width: "min(680px, calc(100vw - 32px))",
  maxHeight: "min(760px, calc(100vh - 32px))",
  flexDirection: "column",
  gap: tokens.space.md,
  overflow: "auto",
  padding: tokens.space.xl,
  border: `1px solid ${tokens.color.borderStrong}`,
  borderRadius: tokens.radius.md,
  backgroundColor: tokens.color.surface,
  color: tokens.color.text,
  boxShadow: `0 20px 60px ${tokens.color.overlay}`,
  transform: "translate(-50%, -50%)",
  animation: `${enter} ${tokens.motion.fast} ease-out`,
});

export const title = style({ margin: 0, fontSize: tokens.fontSize.xl });
export const description = style({ margin: 0, color: tokens.color.textMuted });
export const body = style({
  display: "flex",
  minHeight: 0,
  flexDirection: "column",
  gap: tokens.space.md,
});
export const actions = style({ display: "flex", justifyContent: "flex-end", gap: tokens.space.sm });
