#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <exact-rum-candidate-sha>" >&2
  exit 64
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SOURCE="$ROOT/scripts/ops/rum-judge-owner-review-dev-verifier.sh"
[[ -f "$SOURCE" && ! -L "$SOURCE" ]] || { echo "base Judge owner-review verifier unavailable" >&2; exit 78; }

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT HUP INT TERM
COPY="$work/rum-judge-owner-review-dev-verifier.sh"
cp "$SOURCE" "$COPY"
python3 - "$COPY" <<'PY'
from pathlib import Path
import sys

path=Path(sys.argv[1])
text=path.read_text()

old="sinbin=viewer.get_by_role('button', name='Sin Bin', exact=True); sinbin.click();"
new="sinbin=viewer.locator('button[aria-label=\"Sin Bin\"]'); sinbin.click();"
if old not in text:
    raise SystemExit('strict wrapper blocked: expected Sin Bin verifier selector not found')
text=text.replace(old,new,1)

if 'import os,re\n' not in text:
    raise SystemExit('strict wrapper blocked: verifier Python import marker not found')
text=text.replace('import os,re\n','import os,re,base64\n',1)

start=text.find('def selected_style(locator):')
end=text.find('\nwith sync_playwright()', start)
if start < 0 or end < 0:
    raise SystemExit('strict wrapper blocked: selected-style verifier block not found')
replacement=r'''def selected_style(locator, side):
    return locator.evaluate("""(el, side) => {
      const hit=el.getBoundingClientRect();
      const selected=getComputedStyle(el, '::after');
      const hitStyle=getComputedStyle(el);
      const row=el.closest('.judge-vote-row');
      const image=row.querySelector(':scope > img').getBoundingClientRect();
      const source=side === 'sinbin'
        ? {x0:12, y0:16, x1:169, y1:173}
        : {x0:204, y0:19, x1:353, y1:168};
      const scaleX=image.width / 368;
      const scaleY=image.height / 190;
      const expected={
        left:image.left + source.x0 * scaleX,
        top:image.top + source.y0 * scaleY,
        right:image.left + source.x1 * scaleX,
        bottom:image.top + source.y1 * scaleY,
      };
      const pseudo={
        left:hit.left + parseFloat(selected.left),
        top:hit.top + parseFloat(selected.top),
        width:parseFloat(selected.width),
        height:parseFloat(selected.height),
      };
      pseudo.right=pseudo.left + pseudo.width;
      pseudo.bottom=pseudo.top + pseudo.height;
      const root=getComputedStyle(document.documentElement).getPropertyValue('--brand').trim();
      const probe=document.createElement('span'); probe.style.color=root; document.body.appendChild(probe);
      const brand=getComputedStyle(probe).color; probe.remove();
      return {
        outlineColor:selected.outlineColor,
        outlineWidth:selected.outlineWidth,
        outlineStyle:selected.outlineStyle,
        outlineOffset:selected.outlineOffset,
        borderRadius:selected.borderRadius,
        hitOutlineWidth:hitStyle.outlineWidth,
        hitOutlineStyle:hitStyle.outlineStyle,
        hit:{left:hit.left,top:hit.top,right:hit.right,bottom:hit.bottom,width:hit.width,height:hit.height},
        pseudo,
        expected,
        brand,
      };
    }""", side)

def assert_clean_selected_style(name, style):
    if style['outlineColor'] != style['brand']:
        raise RuntimeError(f"{name} selected outline is not RUM red: {style}")
    if style['outlineWidth'] != '2px' or style['outlineStyle'] != 'solid':
        raise RuntimeError(f"{name} selected outline is not one clean 2px solid accent: {style}")
    if style['outlineOffset'] != '2px':
        raise RuntimeError(f"{name} selected outline does not preserve the required gap: {style}")
    if style['hitOutlineWidth'] != '0px' and style['hitOutlineStyle'] != 'none':
        raise RuntimeError(f"{name} larger hit target still carries a selection outline: {style}")
    deltas={edge:abs(style['pseudo'][edge]-style['expected'][edge]) for edge in ('left','top','right','bottom')}
    if max(deltas.values()) > 1.25:
        raise RuntimeError(f"{name} selected outline geometry does not track the artwork tile: deltas={deltas} style={style}")
    if abs(style['pseudo']['width']-style['pseudo']['height']) > 1.25:
        raise RuntimeError(f"{name} selection target is not rounded-square geometry: {style}")
    margins=[
        style['pseudo']['left']-style['hit']['left'],
        style['hit']['right']-style['pseudo']['right'],
        style['pseudo']['top']-style['hit']['top'],
        style['hit']['bottom']-style['pseudo']['bottom'],
    ]
    if min(margins) < 3:
        raise RuntimeError(f"{name} selection geometry is still too close to the outer hit target: margins={margins} style={style}")
    print(f"{name.lower().replace(' ','_')}_selected_outline={style['outlineWidth']} {style['outlineStyle']} {style['outlineColor']} offset={style['outlineOffset']}")
    print(f"{name.lower().replace(' ','_')}_tile_bounds={style['pseudo']}")
    print(f"{name.lower().replace(' ','_')}_expected_tile_bounds={style['expected']}")
    print(f"{name.lower().replace(' ','_')}_outer_hit_bounds={style['hit']}")
'''
text=text[:start]+replacement+text[end:]

keep_old="keep_style=selected_style(keep); assert_clean_selected_style('Keep', keep_style)"
keep_new="keep_style=selected_style(keep, 'keep'); assert_clean_selected_style('Keep', keep_style); viewer.locator('.judge-vote-row').screenshot(path='/work/judge-keep-selected.png')"
if keep_old not in text:
    raise SystemExit('strict wrapper blocked: Keep style assertion marker not found')
text=text.replace(keep_old, keep_new, 1)

sin_old="sinbin_style=selected_style(sinbin); assert_clean_selected_style('Sin Bin', sinbin_style)"
sin_new="sinbin_style=selected_style(sinbin, 'sinbin'); assert_clean_selected_style('Sin Bin', sinbin_style); viewer.locator('.judge-vote-row').screenshot(path='/work/judge-sinbin-selected.png')"
if sin_old not in text:
    raise SystemExit('strict wrapper blocked: Sin Bin style assertion marker not found')
text=text.replace(sin_old, sin_new, 1)

close_marker="    viewer_context.close(); rater_context.close(); browser.close()"
if close_marker not in text:
    raise SystemExit('strict wrapper blocked: browser close marker not found')
image_dump=r'''    for label in ('keep','sinbin'):
        screenshot_path=f'/work/judge-{label}-selected.png'
        with open(screenshot_path,'rb') as screenshot_file:
            encoded=base64.b64encode(screenshot_file.read()).decode()
        print(f'judge_{label}_selected_png_base64={encoded}')
'''
text=text.replace(close_marker, image_dump+close_marker, 1)

path.write_text(text)
PY
chmod 0700 "$COPY"
printf 'RUM_JUDGE_SINBIN_SELECTOR_COMPAT=VOTE_ARIA_LABEL_ONLY\n'
printf 'RUM_JUDGE_CHECK_AGAIN_ASSERTIONS=STRICT\n'
printf 'RUM_JUDGE_SELECTED_STYLE_ASSERTIONS=TILE_GEOMETRY_AND_SCREENSHOT\n'
bash "$COPY" "$1"
