# PR #169 — YouTube Channel recognition acceptance (isolated DEV)

- Application head: `5939cf2b5a34f74882486c857ed66b1a11f769e1` (PR remains **open / draft / unmerged**).
- Exact-head CI: [43536179781](https://github.com/DaisyCloverSoftware/rum/actions/runs/43536179781), all nine jobs successful.
- Canonical DEV API: `sha256:71ec4ce2fa083270275f475e9a6c7c427088c0ed568263f6abbe5670ca9ba443`.
- RAT frontend: `sha256:9d067ce6244ef9504e67c478d06975c2bf22a1f3c41c1c240c7134b011c15778`.
- Served version: `rat-candidate+5939cf2b5a34f74882486c857ed66b1a11f769e1`.

## Deployed browser screenshots

- [YouTube Channel recognised, **Save new type** enabled without internal kind choice, owner name intact](01-youtube-save-enabled.jpg).
- [Visible label remains **YouTube Channel** after controlled save, original name/description/link intact](03-youtube-selected-label.jpg).
- [Ambiguous **Community channel** asks whether online page or group](04-ambiguous-channel-question.jpg).

## Verified

The genuine isolated RAT owner-form browser acceptance used **The Professional Idiots** as the search-prefilled name. No Thing by that name was submitted. YouTube type-creation responses in this visual test were intercepted; the actual server also returned a pre-existing canonical YouTube Channel classification mapped to `digital_resource`.

An independent separate DEV test journey explicitly created a new persisted classification, a pending Thing, verified a refresh and a fresh form's HTTP catalogue reuse, and completed initial rating. An independent SQL read-back found the primary canonical classification assignment and exactly one rating event for the published test Thing. DJI/CJ, linked navigation and privacy regression browser checks passed.

Supporting original Workbench receipt identifiers and machine-readable outcomes are in [acceptance-report.json](acceptance-report.json). No application rerun, extra migration, reset, reseed, PR merge or LIVE deployment was done for this evidence archive.
