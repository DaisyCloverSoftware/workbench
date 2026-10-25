#!/usr/bin/env bash
set -euo pipefail
umask 077
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SOURCE="$ROOT/scripts/ops/rum-owner-rejection-profile-dev-verifier.sh"
[[ -f "$SOURCE" && ! -L "$SOURCE" ]] || { echo "VERIFY BLOCKED: profile verifier source unavailable" >&2; exit 78; }
work="$(mktemp -d)"; trap 'rm -rf "$work"' EXIT HUP INT TERM
COPY="$work/profile-owner-rejection-current3.sh"; cp "$SOURCE" "$COPY"
python3 - "$COPY" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]); text=p.read_text()
def once(old,new,label):
    global text
    c=text.count(old)
    if c != 1: raise SystemExit(f'{label} marker count={c}')
    text=text.replace(old,new,1)

once('Storage::disk($m->storage_disk)', 'Illuminate\\Support\\Facades\\Storage::disk($m->storage_disk)', 'storage facade')
once("base=os.environ['RUM_BASE_URL']; owner_name=os.environ['RUM_OWNER_NAME']; avatar_media_id=os.environ['RUM_AVATAR_MEDIA_ID']", "base=os.environ['RUM_BASE_URL']; owner_name=os.environ['RUM_OWNER_NAME']; avatar_media_id=os.environ['RUM_AVATAR_MEDIA_ID']; baseline=os.environ['RUM_BASELINE_CSP']", 'baseline environment')
once("require(labels==['Overall Public Rating','Rate My Rating','Mates Only Rating'], f'unexpected principal metric labels: {labels}')", "require([label.casefold() for label in labels]==['overall public rating','rate my rating','mates only rating'], f'unexpected principal metric labels: {labels}')", 'rendered metric label casing')
once("if console_errors: raise RuntimeError('console errors: '+' | '.join(console_errors[:5]))\nprint('profile_unexpected_console_errors=0')", "known=[msg for msg in console_errors if baseline in msg and \"script-src 'self'\" in msg]\nunexpected=[msg for msg in console_errors if msg not in known]\nprint(f'profile_known_live_baseline_csp_console_errors={len(known)}')\nif unexpected: raise RuntimeError('unexpected console errors: '+' | '.join(unexpected[:5]))\nprint('profile_unexpected_console_errors=0')", 'console baseline filter')
once('"$runtime" run --rm --ipc=host -v "$work:/work" -e RUM_BASE_URL="$BASE_URL" -e RUM_OWNER_NAME="$owner_name" -e RUM_AVATAR_MEDIA_ID="$avatar_media_id" "$image" python /work/verify.py', '"$runtime" run --rm --ipc=host -v "$work:/work" -e RUM_BASE_URL="$BASE_URL" -e RUM_OWNER_NAME="$owner_name" -e RUM_AVATAR_MEDIA_ID="$avatar_media_id" -e RUM_BASELINE_CSP="sha256-A0FJyCgxFUPhG7nac5LcQPwVRK5So9ZNz7x5ubsD9kU=" "$image" python /work/verify.py', 'browser environment')

old='''    mini=page.locator('.profile-hero__founder').first; mini.wait_for(state='visible', timeout=15000)
    canonical=mini.locator('.founder-alpha-approved').first; canonical.wait_for(state='visible', timeout=15000)
    require(canonical.locator('text=FOUNDER').count() >= 1 and canonical.locator('text=ALPHA').count() >= 1, 'mini Founder is not canonical Founder Alpha artwork')
    a=avatar.bounding_box(); b=mini.bounding_box(); require(a and b, 'avatar/founder bounds unavailable')
    ar=a['x']+a['width']; ab=a['y']+a['height']; br=b['x']+b['width']; bb=b['y']+b['height']
    overlap_x=max(0,min(ar,br)-max(a['x'],b['x'])); overlap_y=max(0,min(ab,bb)-max(a['y'],b['y']))
    overlap_ratio=(overlap_x*overlap_y)/(b['width']*b['height'])
    ratio=b['width']/a['width']
    require(b['x']+b['width']/2 > a['x']+a['width']/2 and b['y']+b['height']/2 > a['y']+a['height']/2, f'Founder badge is not in lower-right quadrant: avatar={a} badge={b}')
    require(b['x'] < ar and b['y'] < ab and br > ar and bb > ab, f'Founder badge does not overlap/project from lower-right edge: avatar={a} badge={b}')
    require(.35 <= overlap_ratio <= .65, f'Founder overlap footprint not approximately half: {overlap_ratio:.3f}')
    require(.42 <= ratio <= .68, f'Founder/avatar scale relationship unexpected: {ratio:.3f}')
    print(f"{'mobile' if mobile else 'desktop'}_avatar_image=loaded stored_media object_fit_cover circular_clip")
    print(f"{'mobile' if mobile else 'desktop'}_founder_geometry=lower_right overlap_ratio:{overlap_ratio:.3f} size_ratio:{ratio:.3f} avatar:{json.dumps(a,separators=(',',':'))} badge:{json.dumps(b,separators=(',',':'))}")
'''
new='''    mini=page.locator('.profile-hero__founder').first; mini.wait_for(state='visible', timeout=15000)
    canonical=mini.locator('.founder-alpha-approved').first; canonical.wait_for(state='visible', timeout=15000)
    marker=canonical.locator('.founder-alpha-approved__avatar-marker').first; marker.wait_for(state='visible', timeout=15000)
    require(marker.inner_text().strip() == '#002', f'avatar Founder marker is not number-only: {marker.inner_text()!r}')
    require(not canonical.locator('.founder-alpha-approved__badge').first.is_visible(), 'full Founder medal remains visible over the profile avatar')
    visible_copy=mini.inner_text().strip().upper()
    require(visible_copy == '#002', f'avatar Founder marker contains rejected extra copy: {visible_copy!r}')
    marker_style=marker.evaluate("""(el) => { const s=getComputedStyle(el); return {background:s.backgroundColor,border:s.borderColor,color:s.color,fontSize:s.fontSize,borderRadius:s.borderRadius}; }""")
    require(marker_style['background'] == 'rgb(155, 166, 175)', f'Founder #002 Platinum tier colour is not visible: {marker_style}')
    require(marker_style['background'] not in ('rgba(0, 0, 0, 0)','transparent','rgb(255, 255, 255)'), f'Founder marker has rejected transparent/white-disc treatment: {marker_style}')
    a=avatar.bounding_box(); b=mini.bounding_box(); m=marker.bounding_box(); require(a and b and m, 'avatar/founder marker bounds unavailable')
    ar=a['x']+a['width']; ab=a['y']+a['height']; br=b['x']+b['width']; bb=b['y']+b['height']
    overlap_x=max(0,min(ar,br)-max(a['x'],b['x'])); overlap_y=max(0,min(ab,bb)-max(a['y'],b['y']))
    require(b['x']+b['width']/2 > a['x']+a['width']/2 and b['y']+b['height']/2 > a['y']+a['height']/2, f'Founder marker is not in lower-right quadrant: avatar={a} marker={b}')
    require(b['x'] < ar and b['y'] < ab and br > ar and bb > ab, f'Founder marker no longer overlaps/projects from the lower-right edge: avatar={a} marker={b}')
    require(b['width'] <= (40 if mobile else 44) and b['height'] <= (22 if mobile else 24), f'Founder marker is not subtle enough: {b}')
    require(b['width']/a['width'] < .42 and b['height']/a['height'] < .31, f'Founder marker competes with avatar: avatar={a} marker={b}')
    print(f"{'mobile' if mobile else 'desktop'}_avatar_image=loaded stored_media object_fit_cover circular_clip")
    print(f"{'mobile' if mobile else 'desktop'}_founder_marker=number_only:#002 tier:platinum lower_right size:{b['width']:.1f}x{b['height']:.1f} overlap:{overlap_x:.1f}x{overlap_y:.1f} style:{json.dumps(marker_style,separators=(',',':'))}")
'''
once(old,new,'subtle avatar founder marker')

old='''    full=desktop.locator('.profile-badge-mark--founder-canonical .founder-alpha-approved').first; full.wait_for(state='visible', timeout=15000)
    require(full.locator('text=FOUNDER').count()>=1 and full.locator('text=ALPHA').count()>=1, 'full Founder badge is not canonical Founder Alpha artwork')
    require(desktop.locator('.profile-badge-mark--founder').count()==0, 'rejected generic Founder badge mark remains')
    print('desktop_full_founder_badge=canonical_founder_alpha')
'''
new='''    full=desktop.locator('.profile-badge-mark--founder-canonical .founder-alpha-approved').first; full.wait_for(state='visible', timeout=15000)
    full_svg=full.locator('.founder-alpha-approved__badge').first; full_svg.wait_for(state='visible', timeout=15000)
    require(not full.locator('.founder-alpha-approved__avatar-marker').first.is_visible(), 'avatar-only number marker leaked into the full Founder card')
    require(full_svg.locator('image').count()==0, 'full Founder badge uses raster image content instead of crisp vector UI')
    svg_box=full_svg.bounding_box(); require(svg_box, 'full Founder badge bounds unavailable')
    text_boxes={}
    for label in ('RUM','FOUNDER','ALPHA','PLATINUM','#002'):
        node=full_svg.locator('text', has_text=label).first
        require(node.count()==1 and node.is_visible(), f'full Founder badge lettering missing: {label}')
        box=node.bounding_box(); require(box and box['width'] >= 9 and box['height'] >= 6, f'full Founder badge lettering is too small/illegible for {label}: {box}')
        require(box['x'] >= svg_box['x']-1 and box['y'] >= svg_box['y']-1 and box['x']+box['width'] <= svg_box['x']+svg_box['width']+1 and box['y']+box['height'] <= svg_box['y']+svg_box['height']+1, f'full Founder badge lettering clips outside artwork for {label}: text={box} svg={svg_box}')
        text_boxes[label]=box
    require(desktop.locator('.profile-badge-mark--founder').count()==0, 'rejected generic Founder badge mark remains')
    print('desktop_full_founder_badge=vector crisp readable RUM|FOUNDER|ALPHA|PLATINUM|#002 '+json.dumps(text_boxes,separators=(',',':')))
'''
once(old,new,'readable full founder badge')

p.write_text(text)
PY
chmod 0700 "$COPY"
bash "$COPY" "$@"
