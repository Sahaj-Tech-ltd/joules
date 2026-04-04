export const timing = {
  instant: 100,
  fast: 200,
  normal: 300,
  slow: 500,
  verySlow: 800,
} as const;

export type TimingKey = keyof typeof timing;

export const easing = {
  default: [0.25, 0.1, 0.25, 1] as const,
  easeIn: [0.42, 0, 1, 1] as const,
  easeOut: [0, 0, 0.58, 1] as const,
  easeInOut: [0.42, 0, 0.58, 1] as const,
  spring: [0.175, 0.885, 0.32, 1.275] as const,
};

export type EasingKey = keyof typeof easing;
