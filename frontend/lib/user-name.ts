/** Two-letter avatar initials from a full name ("Hamid Rehman" → "HR"). */
export function initials(fullName: string): string {
  const parts = fullName.trim().split(/\s+/).filter(Boolean);
  const first = parts[0]?.charAt(0) ?? "";
  const last =
    parts.length > 1 ? (parts[parts.length - 1]?.charAt(0) ?? "") : "";
  return (first + last).toUpperCase() || "U";
}

/** The given name, falling back to the whole string for single-word names. */
export function firstName(fullName: string): string {
  return fullName.trim().split(/\s+/)[0] || fullName;
}
