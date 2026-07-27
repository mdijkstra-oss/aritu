# Never ask the model for an empty value in a required field

`VerdictSchemaFor` marks both `satisfies` and `reason` required on every unit, with
`additionalProperties: false`. `base.md` used to say of the reason: "A unit that satisfies it
needs none."

That is a contradiction, and the model obeyed the prose: for a satisfying unit it emitted
`{"satisfies": true}` with no `reason`, the CLI rejected the reply against the schema,
resampled, got the same answer, and returned
`terminal_reason=structured_output_retry_exhausted`. aritu reports that as exit 2 — could not
run — so it reads as a broken tool rather than as a contradictory prompt.

**Rewording it to "return an empty string" was not enough.** The failures dropped but did not
stop, and they stayed concentrated in exactly one place: `pass-` fixtures, where *every* unit
satisfies and *every* reason would be `""`. Every `fail-` fixture was stable throughout. One
`pass-` fixture failed on three consecutive runs, survived a process-level retry, and was
still failing after five internal resamples each time.

What fixed it was asking for a short clause on satisfying units too — never for an empty
value. The same fixture then passed three runs in a row with no other change.

Two rules come out of this:

1. **The generated schema is a contract the prose has to honour.** An instruction that a
   field is unnecessary has to be phrased as what to put there instead, because the schema
   has no optional fields.
2. **"Return an empty string" is not a safe way to say that.** A model asked to fill a
   required field with nothing will fight it, and the whole call dies rather than one field.
   Ask for something small and throw it away instead — `collectReasons` already drops the
   reasons of units that reached full agreement, so the text costs nothing downstream.

A related defect found on the way: `answerSchema` emitted `additionalProperties: false` on
the `string` and `boolean` leaves, which the CLI's strict validator rejects. That is fixed —
the keyword now appears only on objects — but it was not what caused the exhaustion.

Worth checking first whenever a rule set produces intermittent exit 2 on fixtures that look
fine: it is likelier to be the prompt disagreeing with the schema than the model failing. See
[[fixtures-hold-only-on-unanimity]].
