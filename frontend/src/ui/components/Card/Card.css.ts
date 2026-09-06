import { globalStyle, style } from "@vanilla-extract/css";
import { tokens } from "../../tokens/contract.css";

export const card = style({
  minWidth: 0,
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.md,
  padding: tokens.space.lg,
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.md,
  backgroundColor: tokens.color.surface,
  color: tokens.color.text,
  boxShadow: tokens.shadow.sm,
});

/** Existing card headings share the mockup's separated header, without changing their semantics. */
globalStyle(`.${card} > h2:first-child`, {
  margin: `calc(-1 * ${tokens.space.lg}) calc(-1 * ${tokens.space.lg}) 0`,
  padding: `${tokens.space.md} ${tokens.space.lg}`,
  borderBottom: `1px solid ${tokens.color.border}`,
  borderRadius: `${tokens.radius.md} ${tokens.radius.md} 0 0`,
  fontSize: tokens.fontSize.md,
  fontWeight: 600,
  letterSpacing: "0.01em",
});
