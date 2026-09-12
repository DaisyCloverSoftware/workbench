# Override Rin field-outfit review capture

This capability is a bounded visual-development evidence path for the current owner-provided Rin wardrobe snapshot. It is not asset adoption, visual acceptance, animation acceptance, an Unreal import path, or a replacement for the released v0.9.62 structural inspection.

## Candidate identity

The review candidate is intentionally distinct from the released structural-inspection candidate.

- protected branch: `agent/hardline-premium-slice-20260822`
- protected HEAD: `3aed03a7d30b987bb76af5fd8230c941a2758e49`
- candidate/stage: `rin-field-outfit-review-v1-eec3287f2544d70f`
- manifest SHA-256: `eec3287f2544d70fd6e132f4c2179790e5f9e5cff90ffea6fc74308c14de5927`
- interface: `host_id` only

The five fixed source records are:

| Role | Relative path | Bytes | SHA-256 |
| --- | --- | ---: | --- |
| integrated outfit | `SourceAssets/RinWardrobe/AST_RL_CHR_RIN_FIELD_OUTFIT.production.blend` | 8731474 | `7ec4d0db94d3f7ad71df357b1e0a801df6a5f873484abddca0803c84c2295469` |
| integrated export | `SourceAssets/RinWardrobe/Exports/SK_RL_Rin_FieldOutfit.fbx` | 5271308 | `115515610f608694791ac377f6e7ad8d3e4fbda1b93700443da6c0fbfb7e7f21` |
| combat boots fit | `SourceAssets/RinWardrobe/FittedBoots/AST_RL_CHR_RIN_BLACK_COMBAT_BOOTS_FIT.blend` | 2206529 | `9fa87f121fdddcf2cee530a46e59cb73fb02bc20e15d0505f9516a35898bb9f6` |
| cargo fit | `SourceAssets/RinWardrobe/FittedCargo/AST_RL_CHR_RIN_FUTURISTIC_CARGO_FIT.blend` | 5753828 | `fdcf2290e86d70c6f80742de4e86e7441693571cbf885a3d7e2939e99619ccbe` |
| cargo authored base | `SourceAssets/RinWardrobe/AuthoredBase/AST_RL_CHR_RIN_CARGO_AUTHORED_BASE.blend` | 3400012 | `306960e889a5e674777f6077d012dc8bc31e181fdd085665487beb52b4f67d10` |

The manifest digest is calculated over those records in the fixed order as newline-delimited `role|relative_path|bytes|sha256` records. Callers cannot provide replacement paths, hashes, roots, scripts, executables, render options, camera positions, or acceptance values.

## Protected-source boundary

Before staging, Workbench requires the exact protected worktree and HEAD and snapshots `git status --porcelain=v1 -z --untracked-files=all` as a SHA-256 plus record count. Each candidate must remain an untracked regular file with the exact fixed size/hash.

Workbench writes only to its own cache. It makes byte-identical copies of the five candidates into the candidate-specific review stage, verifies the source before and after every copy, and verifies the staged copies. It does not checkout, reset, clean, stash, stage, commit, overwrite, regenerate or save through Blender in the protected worktree.

After rendering, all five source records are reverified, protected HEAD is rechecked, and the protected status digest and record count must equal the before snapshot. Otherwise the operation fails.

## Fixed review render

Only the staged integrated outfit blend is opened. Blender is invoked with fixed arguments including:

- `--background`
- `--factory-startup`
- `--disable-autoexec`
- one internally generated fixed Python renderer

The caller cannot supply Python or command-line arguments. Linked external Blend libraries fail closed. External unpacked image dependencies outside the isolated stage are suppressed rather than read from arbitrary local paths. The renderer creates fixed neutral lighting and an orthographic camera and produces exactly three deterministic gross-review views: front convention, three-quarter convention and back convention.

The three views are combined into a single 480x320 JPEG. The JPEG is bounded to 7,500 bytes so its base64 representation and metadata remain inside the existing host-job output limit. Workbench verifies JPEG framing and SHA-256 before returning it. The result contains no local absolute paths.

This small contact sheet is deliberately a first visual-development baseline. It is suitable for judging gross silhouette, fit and whether the current DCC state is worth further iteration. It is not sufficient for final materials, texture fidelity, facial identity, motion quality, cloth simulation, cinematic composition or AAA acceptance.

## Acceptance boundary

A successful review capture must still report:

- `visual_acceptance = not_assessed`
- `animation_acceptance = not_assessed`
- `aaa_acceptance = false`

The image does not approve the wardrobe. It exists so the owner and engineering can see what they are discussing before spending more DCC or Unreal integration effort.

The separate released `override_rin_field_outfit_inspect_v1` contract remains unchanged and continues to have rendering disabled. This review-capture path does not silently repin or widen that released operation.

## Delivery boundary

Source review and CI for this capability do not authorize merge, Workbench version change, release, private-relay update, Windows installation/update, execution against the protected host, asset mutation, Unreal import or asset adoption. Those remain separate decisions.
