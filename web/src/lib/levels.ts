// Levels offered in pickers; A1/A2 are hidden (the learner studies B1 and above).
export const LEVELS = ['B1', 'B2', 'C1', 'C2']

// Keeps a stored level (e.g. the API's A1 default) selectable so a select never shows a blank value.
export const levelOptions = (current: string) => (current && !LEVELS.includes(current) ? [current, ...LEVELS] : LEVELS)
