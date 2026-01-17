 Input Tokens vs Accuracy

  Raw token count alone isn't the issue - it's what those tokens contain:
  ┌───────────────────────────────┬────────────────────────────────┐
  │            Factor             │       Impact on Accuracy       │
  ├───────────────────────────────┼────────────────────────────────┤
  │ More instructions/constraints │ High negative impact           │
  ├───────────────────────────────┼────────────────────────────────┤
  │ More code/data context        │ Low impact (helpful, actually) │
  ├───────────────────────────────┼────────────────────────────────┤
  │ More examples                 │ Usually positive               │
  ├───────────────────────────────┼────────────────────────────────┤
  │ Conflicting instructions      │ Very high negative impact      │
  └───────────────────────────────┴────────────────────────────────┘
  The current tool counts directive patterns (must, never, always) rather than raw tokens because:
  - 50k tokens of code = fine
  - 5k tokens of dense instructions = problematic

  Position Effects ("Lost in the Middle")

  Research shows models pay most attention to:
  1. Beginning of context (system prompt)
  2. End of context (recent messages)
  3. Middle gets less attention - instructions buried in the middle may be missed
