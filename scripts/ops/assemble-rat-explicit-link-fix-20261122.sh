#!/usr/bin/env bash
set -Eeuo pipefail
# Deterministic source edit only. No cluster access, merge, force-push, release or credential reads.
[[ $# -eq 0 ]] || exit 64
branch=fix/rat-persist-explicit-linked-things-20261122
base=695b46bbdd7dfc474c2b8c79ec6dca10b34228b3
t=$(mktemp -d);trap 'rm -rf "$t"' EXIT
git -C "$t" init -q; git -C "$t" remote add origin https://github.com/DaisyCloverSoftware/rum.git; git -C "$t" fetch -q --depth=1 origin "$branch"; git -C "$t" checkout -q --detach FETCH_HEAD; cd "$t"; [[ $(git rev-parse HEAD) == "$base" ]] || exit 78
python3 - <<'PY'
from pathlib import Path
api=Path('apps/api'); src=Path('apps/rate-anything/src')
def change(path, old, new):
 s=path.read_text(); assert s.count(old)==1,(str(path),old[:80],s.count(old)); path.write_text(s.replace(old,new))
p=api/'app/Http/Controllers/Api/V1/StandaloneThingController.php'
change(p,'use App\\Services\\StandaloneThingService;', 'use App\\Services\\StandaloneConnectionIntent;\nuse App\\Services\\StandaloneThingService;')
change(p,'use Illuminate\\Validation\\Rule;', 'use Illuminate\\Support\\Facades\\DB;\nuse Illuminate\\Validation\\Rule;')
change(p,"            'media' => [", "            'connection' => ['sometimes', 'array:sourceId,type,direction'],\n            'connection.sourceId' => ['required_with:connection', 'ulid'],\n            'connection.type' => ['required_with:connection', Rule::in(['related_to', 'manufactured_by', 'located_on'])],\n            'connection.direction' => ['required_with:connection', Rule::in(['outgoing', 'incoming'])],\n            'media' => [")
change(p,'    private function response(', '''    public function retryConnection(Request $request, RatingJourney $journey, StandaloneThingService $service, StandaloneConnectionIntent $connections): JsonResponse
    {
        $actor = $this->actor($request);

        return DB::transaction(function () use ($actor, $request, $journey, $service, $connections): JsonResponse {
            $locked = RatingJourney::query()->where('user_id', $actor->id)->lockForUpdate()->findOrFail((string) $journey->id);
            $target = $service->resume($actor, $locked);
            abort_unless(isset($locked->metadata['connectionIntent']), 404);
            $connections->complete($actor, $locked, $target);

            return $this->response($request, $target, $locked, false);
        }, 3);
    }

    private function response(''')
change(p,"                'created' => $created,", "                'created' => $created,\n                'connection' => $journey?->metadata['connectionIntent'] ?? null,")
p=api/'app/Services/StandaloneThingService.php'
change(p,"        $identity = $input;", "        if (isset($input['connection'])) {\n            $source = Entity::query()->findOrFail((string) $input['connection']['sourceId']);\n            app(RatCommunityContributionPolicy::class)->assertPublicThing($source);\n        }\n        $identity = $input;")
change(p,"                'metadata' => ['standaloneCreation' => [", "                'metadata' => [...(isset($input['connection']) ? ['connectionIntent' => [\n                    'sourceId' => (string) $input['connection']['sourceId'], 'targetId' => (string) $entity->id,\n                    'actorId' => (string) $actor->id, 'type' => $input['connection']['type'],\n                    'direction' => $input['connection']['direction'], 'state' => 'pending',\n                ]] : []), 'standaloneCreation' => [")
p=api/'app/Services/RatingJourneyService.php';change(p,"                    EntityRelationship::query()->whereKey($entry->relationship_id)->update(['visibility' => 'public']);\n                }", "                    EntityRelationship::query()->whereKey($entry->relationship_id)->update(['visibility' => 'public']);\n                }\n                if (isset($journey->metadata['connectionIntent']) && $journey->root_entity_id === $entity->id) {\n                    app(StandaloneConnectionIntent::class)->complete($actor, $journey, $entity);\n                }")
p=api/'app/Services/StandaloneConnectionIntent.php';change(p,"findOrFail($intent['sourceId'])", "findOrFail((string) $intent['sourceId'])")
p=api/'routes/rate-anything-web.php';needle="            Route::get('/standalone-things/{journey}',";change(p,needle,"            Route::post('/standalone-things/{journey}/retry-connection', [StandaloneThingController::class, 'retryConnection'])->middleware('throttle:entity-create')->name('rate-anything.standalone.retry-connection');\n"+needle)
p=src.parent/'nginx.conf';start='    location ~ ^/api/v1/rate-anything/standalone-things/[a-zA-Z0-9]+$ {';s=p.read_text();block=s[s.index(start):s.index('\n    }',s.index(start))+6];change(p,block,block+'\n\n'+block.replace('+$ {','+/retry-connection$ {').replace('limit_except GET','limit_except POST'))
p=src/'standaloneRatingApi.ts';change(p,'export type StandaloneRatingContext = {', '''export type ConnectionChoice = { sourceId: string; type: string; direction: 'incoming' | 'outgoing' }
export type SavedConnection = ConnectionChoice & { targetId: string; state: 'pending' | 'retry' | 'recorded'; relationshipId?: string }
export const retryStandaloneConnection = (journeyId: string) => sessionRequest<StandaloneRatingContext>(
  `/api/v1/rate-anything/standalone-things/${encodeURIComponent(journeyId)}/retry-connection`, { method: 'POST' },
)

export type StandaloneRatingContext = {''');change(p,'meta: { journey: RatingJourneySummary;', 'meta: { connection?: SavedConnection | null; journey: RatingJourneySummary;');change(p,'  description: string\n  idempotencyKey:', '  description: string\n  connection?: ConnectionChoice\n  idempotencyKey:')
p=src/'PersistedAddPanel.tsx';change(p,'createStandaloneThing, type StandaloneCreationInput','createStandaloneThing, type ConnectionChoice, type StandaloneCreationInput');change(p,'  standalone = false,','  standalone = false,\n  connection,');change(p,'  standalone?: boolean','  standalone?: boolean\n  connection?: ConnectionChoice');change(p,'...buildInput(candidateOverride), description:', '...buildInput(candidateOverride), ...(connection ? { connection } : {}), description:');change(p,"{standalone ? 'Add a Thing' : 'Add to RUM'}", "{connection ? 'Add a linked Thing' : standalone ? 'Add a Thing' : 'Add to RUM'}");change(p,'New Things remain private until their initial rating is complete.', "New Things remain private until their initial rating is complete. {connection ? 'Your requested connection will be saved when the Thing is published.' : ''}")
p=src/'RatCommunityConnectionPanel.tsx';change(p,'  onCreateMissing: (name: string) => void',"  onCreateMissing: (name: string, type: string, direction: 'incoming' | 'outgoing') => void");change(p,"useState('related_to')", "useState(() => new URLSearchParams(window.location.search).get('ct') || 'related_to')");change(p,"useState<'outgoing' | 'incoming'>('outgoing')", "useState<'outgoing' | 'incoming'>(() => new URLSearchParams(window.location.search).get('cd') === 'incoming' ? 'incoming' : 'outgoing')");change(p,'onCreateMissing(query.trim())','onCreateMissing(query.trim(), relationship, direction)');s=p.read_text();selectors=s[s.index('        <label><span>Connection type'):s.index('        <button type="button" className="primary"')];change(p,selectors,'');change(p,'      {performed ? <div', '      {performed || selected ? <div className="rat-community-choice">\n'+selectors.replace('{selected.name}',"{selected?.name ?? 'the linked Thing'}")+'      </div> : null}\n      {performed ? <div')
p=src/'RatEntityWorkspace.tsx';change(p,"import { RatCommunityConnectionPanel }", "import { ratCommunityConnectionApi } from './ratCommunityConnectionApi'\nimport type { ConnectionChoice } from './standaloneRatingApi'\nimport { RatCommunityConnectionPanel }");change(p,'  const [resumeTarget,', "  const [linkChoice, setLinkChoice] = useState<ConnectionChoice>({ sourceId: entityId, type: 'related_to', direction: 'outgoing' })\n  const [linkMessage, setLinkMessage] = useState('')\n  const [resumeTarget,");change(p,"    params.delete('connect')", "    params.delete('connect')\n    params.delete('ct')\n    params.delete('cd')\n    setLinkMessage('Connection saved.')");change(p,'  const createdMissing = (','  const createdMissing = async (');change(p,"+ '~' + encodeURIComponent(entityId) + '?connect=' + encodeURIComponent(createdId)","+ '~' + encodeURIComponent(entityId) + '#connections'");change(p,'      setResumeTarget(createdId)', '''      // A resolved existing candidate has no publication journey. Save once on this
      // explicit action, never as a side effect of opening a URL or matching a name.
      const params = new URLSearchParams({ connect: createdId, ct: linkChoice.type, cd: linkChoice.direction })
      window.history.replaceState({}, '', `${window.location.pathname}?${params}#connections`)
      try {
        await ratCommunityConnectionApi.connect(entityId, createdId, linkChoice.type, linkChoice.direction)
        connected()
      } catch {
        setResumeTarget(createdId)
        setLinkMessage('The Thing exists, but the connection was not confirmed. Retry Confirm connection below; do not create or rate it again.')
      }''');change(p,'onCreateMissing={setMissingName}', '''onCreateMissing={(name, type, direction) => {
                setLinkChoice({ sourceId: active.id, type, direction }); setLinkMessage(''); setMissingName(name)
              }}''');change(p,'            {connectionOpen ? <RatCommunityConnectionPanel', '            {linkMessage ? <p role="status">{linkMessage}</p> : null}\n            {connectionOpen ? <RatCommunityConnectionPanel');change(p,'standalone initialName={missingName}', 'standalone connection={linkChoice} initialName={missingName}')
p=src/'RateAnythingApp.tsx';change(p,'<RatEntityWorkspace entityId=', '<RatEntityWorkspace key={entityId} entityId=')
p=src/'RatRatingExperience.tsx';change(p,"import { resumeStandaloneRating }", "import { resumeStandaloneRating, retryStandaloneConnection, type SavedConnection }");change(p,'  const [publicationMessage,', "  const [connection, setConnection] = useState<SavedConnection | null>(null)\n  const [retrying, setRetrying] = useState(false)\n  const [publicationMessage,");s=p.read_text();assert s.count('setPublication(response.data.publicationState)')==2;p.write_text(s.replace('setPublication(response.data.publicationState)','setPublication(response.data.publicationState)\n          setConnection(response.meta.connection ?? null)'));change(p,'  const leaveRating = () => {', '''  const retryConnection = async () => {
    if (!journeyId || retrying) return
    setRetrying(true)
    try {
      const result = await retryStandaloneConnection(journeyId)
      setConnection(result.meta.connection ?? null)
      setPublicationMessage('')
    } catch {
      setPublicationMessage('Your Thing was saved. The connection could not be confirmed. Retry the connection only.')
    } finally { setRetrying(false) }
  }

  const leaveRating = () => {''');change(p,'        <button className="outline-button" type="button" onClick={leaveRating}>Back to RAT</button>', '''        {connection ? <div role="status">
          {connection.state === 'recorded' ? 'Connection saved.' : connection.state === 'pending' ? 'Connection pending initial publication.' : 'The Thing was published but the connection was not saved.'}
          {connection.state === 'retry' ? <button type="button" className="outline-button" disabled={retrying} onClick={() => void retryConnection()}>Retry connection only</button> : null}
        </div> : null}
        <button className="outline-button" type="button" onClick={leaveRating}>{connection ? 'Return to connected Things' : 'Back to RAT'}</button>''')
PY
git diff --check; git diff --stat
# Push only the isolated branch; non-fast-forward is rejected. No default or stack parent can change.
git add apps/api apps/rate-anything;git -c user.name='Workbench' -c user.email='workbench-relay@users.noreply.github.com' commit -qm 'Persist explicit RAT linking intent through standalone publication and retry'; git push origin "HEAD:refs/heads/$branch";git rev-parse HEAD
