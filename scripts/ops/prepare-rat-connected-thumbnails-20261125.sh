#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 0 ]] || exit 64
base=2359178af19fdd9b5b45d103c892cdc36c2ad086;branch=fix/rat-connected-profile-thumbnails-20261125;t=$(mktemp -d);trap 'rm -rf "$t"' EXIT
[[ $(git ls-remote https://github.com/DaisyCloverSoftware/rum.git refs/heads/feat/rat-shared-persona-discovery-20261124|cut -f1) == "$base" ]]||exit 78
[[ -z $(git ls-remote https://github.com/DaisyCloverSoftware/rum.git "refs/heads/$branch") ]]||{ echo EXISTING_BRANCH_RECOVER_DO_NOT_OVERWRITE;exit 78; }
git -C "$t" init -q;git -C "$t" remote add origin https://github.com/DaisyCloverSoftware/rum.git;git -C "$t" fetch -q --depth=1 origin "$base";git -C "$t" checkout -q --detach FETCH_HEAD
python3 - "$t" <<'PY'
from pathlib import Path
import sys
r=Path(sys.argv[1]);src=r/'apps/rate-anything/src'
p=src/'SearchExperience.tsx';s=p.read_text();start=s.index('function PublishedProfileThumbnail(');end=s.index('\nfunction LinkedCard',start);s=s[:start]+s[end:];s="import { PublishedProfileThumbnail } from './PublishedProfileThumbnail'\n"+s;p.write_text(s)
(src/'PublishedProfileThumbnail.tsx').write_text('''import { useState } from 'react'
import type { RatEntity } from './ratExplorerApi'

/** Only render image information already authorised by the canonical public API. */
export function PublishedProfileThumbnail({ entity, className, fallback }: {
  entity: Pick<RatEntity, 'id' | 'name' | 'images'>
  className: string
  fallback?: string
}) {
  const [failedImage, setFailedImage] = useState<string | null>(null)
  const profileUrl = entity.images?.profile?.status === 'published' ? entity.images.profile.url : null
  const imageKey = `${entity.id}:${profileUrl ?? ''}`

  return <span className={className} aria-hidden="true">
    {profileUrl && failedImage !== imageKey
      ? <img src={profileUrl} alt="" draggable={false} onError={() => setFailedImage(imageKey)}/>
      : (fallback ?? entity.name.slice(0, 2).toUpperCase())}
  </span>
}
''')
p=src/'RatEntityWorkspace.tsx';s=p.read_text();old='<span className="rat-workspace-connection__mark" aria-hidden="true">{initials(item.name)}</span>';assert s.count(old)==1;s=s.replace(old,'<PublishedProfileThumbnail entity={item} className="rat-workspace-connection__mark" fallback={initials(item.name)}/>');p.write_text("import { PublishedProfileThumbnail } from './PublishedProfileThumbnail'\n"+s)
p=src/'RatRelatedConnections.tsx';s=p.read_text();old='<span className="rat-workspace-connection__mark" aria-hidden="true">{item.name.split(/\\s+/).filter(Boolean).slice(0, 2).map((part) => part[0]).join(\'\').toUpperCase()}</span>';assert s.count(old)==1;s=s.replace(old,'<PublishedProfileThumbnail entity={item} className="rat-workspace-connection__mark" fallback={item.name.split(/\\s+/).filter(Boolean).slice(0, 2).map((part) => part[0]).join(\'\').toUpperCase()}/>');p.write_text("import { PublishedProfileThumbnail } from './PublishedProfileThumbnail'\n"+s)
css=[p for p in src.glob('*.css') if '.rat-workspace-connection__mark {' in p.read_text()];assert len(css)==1;p=css[0];s=p.read_text();old='.rat-workspace-connection__mark {';s=s.replace(old,old+' overflow: hidden;',1);s+='\n.rat-workspace-connection__mark img { display: block; width: 100%; height: 100%; object-fit: cover; }\n';p.write_text(s)
(src/'publishedProfileThumbnail.test.ts').write_text('''import { readFileSync } from 'node:fs'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import { PublishedProfileThumbnail } from './PublishedProfileThumbnail'
import { RatRelatedConnections } from './RatRelatedConnections'
import type { RatEntity, RatEntityImage, RatRelatedGroup } from './ratExplorerApi'

const image: RatEntityImage = { url: '/public/profile.jpg', status: 'published', sourceType: 'uploaded', revisionId: 'published-revision', updatedAt: null, sourceSynchronised: false }
const entity: Pick<RatEntity, 'id' | 'name' | 'images'> = { id: 'channel', name: 'Fixture Channel', images: { profile: image, cover: null } }
const render = (value = entity, fallback?: string) => renderToStaticMarkup(createElement(PublishedProfileThumbnail, { entity: value, className: 'existing-size', fallback }))

describe('shared authorised published-profile thumbnail', () => {
  it('renders the published profile, keeping its existing wrapper and decorative semantics', () => {
    const html = render()
    expect(html).toContain('class="existing-size" aria-hidden="true"')
    expect(html).toContain('src="/public/profile.jpg"')
    expect(html).toContain('alt=""')
    expect(html).not.toContain('>FI<')
  })
  it.each(['pending', 'private', 'rejected', 'unpublished'])('does not render a %s profile even if a URL is present', (status) => {
    const html = render({ ...entity, images: { profile: { ...image, status } as RatEntityImage, cover: image } })
    expect(html).not.toContain('<img')
    expect(html).not.toContain('/public/profile.jpg')
    expect(html).toContain('>FI</span>')
  })
  it('falls back without substituting the cover or an external source', () => {
    expect(render({ ...entity, images: undefined })).toContain('>FI</span>')
    const html = render({ ...entity, images: { profile: null, cover: image } }, 'FC')
    expect(html).toContain('>FC</span>')
    expect(html).not.toContain('<img')
    expect(render({ ...entity, images: { profile: { ...image, url: '' }, cover: null } })).not.toContain('<img')
  })
  it('renders a published thumbnail on the actual indirect card without changing canonical navigation or path labels', () => {
    const step = { id: 'edge-one', sourceId: 'source', targetId: 'via', type: 'related_to', label: 'Related to', direction: 'outgoing' as const, verificationState: 'unverified', provenance: 'user_submitted' }
    const groups: RatRelatedGroup[] = [{ key: 'channels:via', label: 'Other channels connected through Creator', via: { id: 'via', name: 'Creator', slug: null }, total: 1, items: [{ ...entity, slug: null, description: null, entityType: 'digital_resource', entityTypeLabel: 'Channel', rateability: 'open', classifications: [], reputation: null, ratingOwner: 'rat', path: [step, { ...step, id: 'edge-two', sourceId: 'via', targetId: entity.id }] }] }]
    const html = renderToStaticMarkup(createElement(RatRelatedConnections, { groups, onOpen: () => {} }))
    expect(html).toContain('data-entity-id="channel"')
    expect(html).toContain('src="/public/profile.jpg"')
    expect(html).toContain('not direct links or verified shared ownership')
  })
  it('uses the same published/failure guard for direct, indirect and search cards', () => {
    for (const filename of ['RatEntityWorkspace.tsx', 'RatRelatedConnections.tsx', 'SearchExperience.tsx']) {
      const source = readFileSync(new URL(`./${filename}`, import.meta.url), 'utf8')
      expect(source).toContain("import { PublishedProfileThumbnail } from './PublishedProfileThumbnail'")
      expect(source).toContain('<PublishedProfileThumbnail entity={item}')
    }
    const source = readFileSync(new URL('./PublishedProfileThumbnail.tsx', import.meta.url), 'utf8')
    expect(source).toContain('onError={() => setFailedImage(imageKey)}')
    expect(source).toContain('failedImage !== imageKey')
  })
})
''')
PY
cd "$t/apps/rate-anything";npm ci --ignore-scripts --no-audit --no-fund >"$t/npm.log" 2>&1
npm run lint; npm run typecheck; npm test -- src/publishedProfileThumbnail.test.ts src/ratRelatedConnections.test.ts; npm run build
cd "$t";git diff --check;git add apps/rate-anything/src;git -c user.name=Workbench -c user.email=workbench@users.noreply.github.com commit -m 'Show authorised published profile thumbnails on connected Things';git push origin "HEAD:refs/heads/$branch"
echo "THUMBNAIL_SOURCE_READY sha=$(git rev-parse HEAD) branch=$branch base=$base";git diff --stat "$base" HEAD
