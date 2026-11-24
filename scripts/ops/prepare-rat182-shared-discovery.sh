#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 0 ]] || exit 64
branch=feat/rat-shared-persona-discovery-20261124
expected=4e9567b09847662d07563d834dd2d6163b946da5
base=b135949c8d97072cca3b24c2611984ae6d6fc30c
ns=rum-rate-anything-preview; pod=rat182-source-preflight-r1; made=false; uid=''
t=$(mktemp -d); umask 077
k(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
cleanup(){ set +e; if $made && [[ $(k -n "$ns" get pod "$pod" -o 'jsonpath={.metadata.uid}' 2>/dev/null) == "$uid" ]]; then k -n "$ns" delete pod "$pod" --wait=false; fi; rm -rf "$t"; }
trap cleanup EXIT
mkdir "$t/source"; git -C "$t/source" init -q; git -C "$t/source" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$t/source" fetch -q --depth=8 origin "$branch"; git -C "$t/source" checkout -q --detach FETCH_HEAD
cd "$t/source"; [[ $(git rev-parse HEAD) == "$expected" ]] || { echo SOURCE_HEAD_MOVED; exit 78; }; git merge-base --is-ancestor "$base" HEAD
python3 - <<'PY'
from pathlib import Path

def patch(path, old, new):
 p=Path(path); s=p.read_text(); assert s.count(old)==1,(path,old[:80],s.count(old)); p.write_text(s.replace(old,new))
p='apps/api/app/Services/RatExplorerService.php'
patch(p,'        private readonly PublicEntityReputationSummaryService $reputation,','        private readonly PublicEntityReputationSummaryService $reputation,\n        private readonly RatSharedConnectionDiscovery $sharedConnections,')
patch(p,"            'categories' => $categoryList,", "            'categories' => $categoryList,\n            'relatedThrough' => $this->relatedThrough(\n                $active, array_map(static fn (array $item): string => (string) $item['entity']->id, $linked), $request,\n            ),")
method='''    /**
     * @param  list<string>  $directIds
     * @return list<array<string,mixed>>
     */
    private function relatedThrough(Entity $active, array $directIds, Request $request): array
    {
        $groups = $this->sharedConnections->discover($active, $directIds);
        $ids = [];
        foreach ($groups as $group) {
            foreach ($group['items'] as $item) {
                $ids[] = (string) $item['entity']->id;
            }
        }
        if ($ids === []) {
            return [];
        }
        $summaries = $this->reputation->forEntityIds($ids);
        $result = [];
        foreach ($groups as $group) {
            $items = [];
            foreach ($group['items'] as $item) {
                $items[] = [
                    ...$this->summary($item['entity'], $summaries[(string) $item['entity']->id] ?? null, $request),
                    'path' => $item['path'],
                ];
            }
            $result[] = [
                'key' => $group['key'], 'label' => $group['label'],
                'via' => ['id' => (string) $group['via']->id, 'name' => (string) $group['via']->canonical_name, 'slug' => $group['via']->canonical_slug],
                'total' => count($items), 'items' => $items,
            ];
        }

        return $result;
    }

'''
patch(p,'    /** @return list<array<string,mixed>> */\n    private function relationships',method+'    /** @return list<array<string,mixed>> */\n    private function relationships')
p='apps/rate-anything/src/ratExplorerApi.ts'
types='''export type RatConnectionStep = RatLinkedItem['relationship'] & {
  sourceId: string
  targetId: string
  provenance: string
}

export type RatRelatedGroup = {
  key: string
  label: string
  via: Pick<RatEntity, 'id' | 'name' | 'slug'>
  total: number
  items: Array<RatEntity & { path: [RatConnectionStep, RatConnectionStep] }>
}

'''
patch(p,'export type RatExplorerNode = {', types+'export type RatExplorerNode = {')
patch(p,'  categories: RatCategory[]\n}', '  categories: RatCategory[]\n  relatedThrough?: RatRelatedGroup[]\n}')
p='apps/rate-anything/src/RatEntityWorkspace.tsx'
patch(p,"import { RatCommunityConnectionPanel }", "import { RatRelatedConnections } from './RatRelatedConnections'\nimport { RatCommunityConnectionPanel }")
patch(p,'const openConnection = (item: RatLinkedItem)', "const openConnection = (item: Pick<RatEntity, 'id' | 'slug'>)")
patch(p,'<span>{connectionCount} public</span>', '<span aria-label="Direct public connections">{connectionCount} direct public</span>')
patch(p,'className="rat-workspace-connection-group" key={category.key}', 'className="rat-workspace-connection-group" data-connection-kind="direct" key={category.key}')
patch(p,'            <button type="button" className="outline-button" aria-expanded={connectionOpen}', '            <RatRelatedConnections groups={node.relatedThrough ?? []} onOpen={openConnection}/>\n            <button type="button" className="outline-button" aria-expanded={connectionOpen}')
patch(p,'<dt>Public connections</dt>', '<dt>Direct public connections</dt>')
p='apps/api/app/Services/RatSharedConnectionDiscovery.php'
patch(p,'use App\\Models\\EntityClassification;\n','')
patch(p,'                $viaId = (string) $via->id;', "                if (! $via instanceof Entity || $entry->type === null) {\n                    continue;\n                }\n                $viaId = (string) $via->id;")
patch(p,'                    $targetId = (string) $target->id;', "                    if (! $target instanceof Entity || $exit->type === null) {\n                        continue;\n                    }\n                    $targetId = (string) $target->id;")
patch(p,'private function other(EntityRelationship $edge, Entity $from): Entity','private function other(EntityRelationship $edge, Entity $from): ?Entity')
patch(p,"(string) $edge->type->key", "(string) $edge->type?->key")
patch(p,"$edge->type->forward_label : $edge->type->reverse_label", "$edge->type?->forward_label : $edge->type?->reverse_label")
p=Path(p); s=p.read_text(); a=s.index('        $entity->loadMissing(\'classificationAssignments.classification\');',s.index('private function isChannel')); b=s.index('\n    }',a)
s=s[:a]+'''        return $entity->classificationAssignments()
            ->where('is_primary', true)->whereNull('valid_to')->where('valid_from', '<=', now())
            ->whereHas('classification', fn (Builder $classification) => $classification
                ->where('status', 'active')->whereIn('key', $this->channelKeys()))->exists();'''+s[b:];p.write_text(s)
p='apps/api/tests/Feature/Entity/RatSharedConnectionDiscoveryTest.php'
patch(p,"        $this->node($persona)->assertJsonCount(0, 'data.relatedThrough')->assertJsonPath('data.categories.0.total', 2);", "        $personData = $this->node($persona)->assertJsonCount(0, 'data.relatedThrough')->json('data');\n        self::assertSame(2, collect($personData['categories'])->sum('total'));")
PY
(cd apps/rate-anything; npm ci --no-audit --no-fund --ignore-scripts >"$t/npm" 2>&1 || { tail -35 "$t/npm"; exit 1; }; npm run build >"$t/build" 2>&1 || { tail -45 "$t/build"; exit 1; }; npm run lint >"$t/lint" 2>&1 || { tail -45 "$t/lint"; exit 1; }; npm test -- src/ratRelatedConnections.test.ts >"$t/ui" 2>&1 || { tail -70 "$t/ui"; exit 1; }; tail -12 "$t/ui")
echo RAT182_FRONTEND_PREFLIGHT_PASS
[[ $(k -n "$ns" get ingress -o 'jsonpath={.items[*].spec.rules[*].host}') == dev-rum-ra.daisycloversoftware.uk ]] || exit 78
if k -n "$ns" get pod "$pod" >/dev/null 2>&1; then echo EXISTING_TEST_POD_STOP; exit 78; fi
k -n "$ns" get deployment rum-rate-anything -o json | python3 -c 'import sys,json,secrets,base64
pulls=json.load(sys.stdin)["spec"]["template"]["spec"].get("imagePullSecrets",[]);pw=secrets.token_hex(24)
def key():return "base64:"+base64.b64encode(secrets.token_bytes(32)).decode()
def env(d):return [{"name":k,"value":v} for k,v in d.items()]
ae={"APP_ENV":"testing","APP_KEY":key(),"APP_DEBUG":"false","DB_CONNECTION":"pgsql","DB_HOST":"127.0.0.1","DB_DATABASE":"rum_test","DB_USERNAME":"fixture","DB_PASSWORD":pw,"CACHE_STORE":"array","SESSION_DRIVER":"array","QUEUE_CONNECTION":"sync","MAIL_MAILER":"array","MODERATION_EVIDENCE_KEY":key(),"MODERATION_AUDIT_IP_HASH_KEY":key(),"HOME":"/tmp","COMPOSER_HOME":"/tmp/composer"}
def security(uid):return {"runAsUser":uid,"runAsGroup":uid,"allowPrivilegeEscalation":False,"capabilities":{"drop":["ALL"]}}
cs=[{"name":"api","image":"ghcr.io/daisycloversoftware/rum-api@sha256:8f22ae6cbe62f06e3363cf3ec897d769c59e4e6522d87f919a62f443943ac894","command":["sleep","1200"],"workingDir":"/work","securityContext":security(10001),"env":env(ae),"volumeMounts":[{"name":"work","mountPath":"/work"}],"resources":{"requests":{"cpu":"100m","memory":"128Mi"},"limits":{"cpu":"2","memory":"1536Mi"}}},{"name":"pg","image":"postgres@sha256:eb63372c1e05c755f55fa30cbd07aecc443676fc6c0fa90b9002d01537841e1a","args":["postgres","-c","listen_addresses=127.0.0.1"],"securityContext":security(999),"env":env({"POSTGRES_DB":"rum_test","POSTGRES_USER":"fixture","POSTGRES_PASSWORD":pw}),"volumeMounts":[{"name":"db","mountPath":"/var/lib/postgresql"}],"readinessProbe":{"exec":{"command":["pg_isready","-h","127.0.0.1","-U","fixture","-d","rum_test"]},"periodSeconds":2},"resources":{"requests":{"cpu":"80m","memory":"96Mi"},"limits":{"cpu":"1","memory":"512Mi"}}}]
print(json.dumps({"apiVersion":"v1","kind":"Pod","metadata":{"name":sys.argv[1],"labels":{"task":"rat182-source-tests"}},"spec":{"restartPolicy":"Never","activeDeadlineSeconds":1200,"terminationGracePeriodSeconds":3,"automountServiceAccountToken":False,"enableServiceLinks":False,"securityContext":{"fsGroup":999,"runAsNonRoot":True,"seccompProfile":{"type":"RuntimeDefault"}},"imagePullSecrets":pulls,"containers":cs,"volumes":[{"name":"work","emptyDir":{"sizeLimit":"2Gi"}},{"name":"db","emptyDir":{"sizeLimit":"512Mi"}}]}}))' "$pod" | k -n "$ns" create -f -
made=true; uid=$(k -n "$ns" get pod "$pod" -o 'jsonpath={.metadata.uid}')
k -n "$ns" wait --for=condition=Ready pod/"$pod" --timeout=180s
# Copy source only, including the narrowly patched working files. No environment, credentials or mounted application data.
git ls-files apps/api | tar -cf - --files-from=- | k -n "$ns" exec -i "$pod" -c api -- tar -xf - -C /work --strip-components=2 --no-same-owner
k -n "$ns" exec "$pod" -c api -- sh -ec 'test "$APP_ENV" = testing && test "$DB_HOST" = 127.0.0.1 && test "$DB_DATABASE" = rum_test; php -r '\''$p=new PDO("pgsql:host=127.0.0.1;dbname=rum_test",getenv("DB_USERNAME"),getenv("DB_PASSWORD"));if($p->query("SELECT count(*) FROM information_schema.tables WHERE table_schema=\x27public\x27")->fetchColumn()!=0)exit(78);'\''; composer install --no-interaction --prefer-dist --no-progress >/tmp/install.log 2>&1 || { tail -40 /tmp/install.log;exit 1; }; vendor/bin/pint app/Services/RatExplorerService.php app/Services/RatSharedConnectionDiscovery.php tests/Feature/Entity/RatSharedConnectionDiscoveryTest.php; vendor/bin/phpstan analyse --no-progress --memory-limit=1G >/tmp/phpstan.log 2>&1 || { cat /tmp/phpstan.log;exit 1; }; cat /tmp/phpstan.log; vendor/bin/phpunit --filter RatSharedConnectionDiscoveryTest >/tmp/phpunit.log 2>&1 || { tail -130 /tmp/phpunit.log;exit 1; }; cat /tmp/phpunit.log'
k -n "$ns" exec "$pod" -c api -- tar -cf - app/Services/RatExplorerService.php app/Services/RatSharedConnectionDiscovery.php tests/Feature/Entity/RatSharedConnectionDiscoveryTest.php | tar -xf - -C apps/api --no-same-owner
git diff --check; git add apps/api/app/Services/RatExplorerService.php apps/api/app/Services/RatSharedConnectionDiscovery.php apps/api/tests/Feature/Entity/RatSharedConnectionDiscoveryTest.php apps/rate-anything/src/ratExplorerApi.ts apps/rate-anything/src/RatEntityWorkspace.tsx
git -c user.name='Workbench' -c user.email='workbench@users.noreply.github.com' commit -m 'Wire read-only shared connections into RAT and validate guarded two-hop paths'
git push origin HEAD:"$branch"
echo "RAT182_IMPLEMENTED_AND_PREFLIGHT_PASS source=$(git rev-parse HEAD) base=$base branch=$branch canonical_writes=0"
git diff --stat "$base" HEAD
