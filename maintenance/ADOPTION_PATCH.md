# Adoption-hardening correction

The Go lexer inherited signed integer parsing for the four hexadecimal digits
after a JSONata string's `\u` escape. It therefore accepted `\u+001` and
`\u-001`, which are not four hexadecimal digits. JavaScript already rejects
both with S0104.

The maintained correction uses unsigned, 16-bit parsing for both the primary
code unit and surrogate lookahead. Valid escape handling is unchanged. The
native lexer table, `contract/cases/adoption-escape-cases.json`, and the
JavaScript adoption-escape tests retain the cross-runtime witnesses. The audit
records the failing-before Go run and passing corrected run.

This is a correctness repair, not a numerical-policy change, language extension,
or new public executor API. Historical upstream identities remain recorded in
their existing provenance files. A future upstream refresh must retain this
witness or demonstrate that upstream now rejects the same malformed input.
