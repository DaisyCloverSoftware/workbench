#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 0 ]] || exit 64
ops=$(cd "$(dirname "$0")" && pwd);t=$(mktemp -d);trap 'rm -rf "$t"' EXIT
python3 - "$ops/prepare-rat182-shared-discovery.sh" "$t/repair.sh" <<'MAKE'
from pathlib import Path
import sys
s=Path(sys.argv[1]).read_text()
for old,new in [
 ('expected=4e9567b09847662d07563d834dd2d6163b946da5','expected=b8127ac50dce516379c6a188403d2a99277cfce6'),
 ('pod=rat182-source-preflight-r1','pod=rat182-public-status-preflight-r1'),
 ('8f22ae6cbe62f06e3363cf3ec897d769c59e4e6522d87f919a62f443943ac894','6ad2db06b44acca08d26ac4e062bc77d279513e45a62e862d4bb85e10a21d450'),
 ('Wire read-only shared connections into RAT and validate guarded two-hop paths','Respect published unverified canonical status in shared discovery'),
 ('RAT182_IMPLEMENTED_AND_PREFLIGHT_PASS','RAT182_PUBLIC_STATUS_REPAIR_PREFLIGHT_PASS')]:
 assert s.count(old)==1,(old,s.count(old));s=s.replace(old,new)
a=s.index("python3 - <<'PY'\n");b=s.index('\nPY\n(cd apps/rate-anything;',a)+len('\nPY')
patch=r'''python3 - <<'PATCH'
from pathlib import Path
p=Path('apps/api/app/Services/RatSharedConnectionDiscovery.php');s=p.read_text()
old="$active->status !== 'active'";assert s.count(old)==1;s=s.replace(old,"in_array($active->status, ['deleted', 'merged'], true)")
old="$query->where('entities.status', 'active')";assert s.count(old)==1;s=s.replace(old,"$query->whereNotIn('entities.status', ['deleted', 'merged'])")
old='        $this->policy->constrain($query);';assert s.count(old)==1;s=s.replace(old,"        // Match UniversalEntityReadService: public unverified is not deleted or merged.\n"+old)
p.write_text(s)
p=Path('apps/api/tests/Feature/Entity/RatSharedConnectionDiscoveryTest.php');s=p.read_text()
marker='    public function test_duplicate_paths_direct_neighbours_and_cycles_never_duplicate_results_or_recurse(): void'
new="""    /** @return iterable<string,array{string,string}> */
    public static function publicStatuses(): iterable
    {
        yield 'active channels' => ['active', 'active'];
        yield 'unverified source' => ['unverified', 'active'];
        yield 'unverified destination' => ['active', 'unverified'];
    }

    #[DataProvider('publicStatuses')]
    public function test_published_unverified_person_status_is_not_mistaken_for_a_moderation_ban(string $sourceStatus, string $targetStatus): void
    {
        [$first, $persona, $second] = $this->graph();
        $persona->forceFill([
            'entity_type_id' => EntityType::query()->where('key', 'person')->firstOrFail()->id,
            'status' => 'unverified',
        ])->save();
        $first->forceFill(['status' => $sourceStatus])->save();
        $second->forceFill(['status' => $targetStatus])->save();
        $before = $this->snapshot();

        foreach ([[$first, $second], [$second, $first]] as [$source, $target]) {
            $this->node($source)->assertOk()
                ->assertJsonPath('data.categories.0.total', 1)
                ->assertJsonCount(1, 'data.relatedThrough')
                ->assertJsonPath('data.relatedThrough.0.via.id', $persona->id)
                ->assertJsonPath('data.relatedThrough.0.items.0.id', $target->id)
                ->assertJsonPath('data.relatedThrough.0.items.0.path.0.verificationState', 'unverified')
                ->assertJsonPath('data.relatedThrough.0.items.0.path.1.verificationState', 'unverified');
        }
        self::assertSame('unverified', $persona->fresh()->status);
        self::assertSame($before, $this->snapshot());

        // Unverified is not blanket eligibility: ordinary RUM people stay private.
        $persona->forceFill(['sensitivity' => 'normal', 'rateability' => 'closed'])->save();
        $this->node($first)->assertOk()->assertJsonCount(0, 'data.relatedThrough')
            ->assertJsonMissing(['id' => $persona->id]);
    }

"""
assert s.count(marker)==1;p.write_text(s.replace(marker,new+marker))
PATCH'''
s=s[:a]+patch+s[b:]
canary=r'''
# Validate the new read-only query against the exact existing canonical records.
# This uses a separate PHP process and a renamed in-memory class; no file in the running pod is edited.
[[ $(k -n rum-dev-isolated get ingress -o 'jsonpath={.items[*].spec.rules[*].host}') == dev-rum.daisycloversoftware.uk ]] || exit 78
[[ $(k -n rum-dev-isolated get deploy rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}') == ghcr.io/daisycloversoftware/rum-api@sha256:6ad2db06b44acca08d26ac4e062bc77d279513e45a62e862d4bb85e10a21d450 ]] || exit 78
python3 - apps/api/app/Services/RatSharedConnectionDiscovery.php "$t/canary.php" <<'CANARY'
from pathlib import Path
import sys
s=Path(sys.argv[1]).read_text();head='<?php\n\nnamespace App\\Services;\n';assert s.startswith(head);s=s[len(head):];assert s.count('final class RatSharedConnectionDiscovery')==1;s=s.replace('final class RatSharedConnectionDiscovery','final class Rat182PublicStatusReadOnlyProbe')
bootstrap=r"""<?php
namespace {
require 'vendor/autoload.php';$app=require 'bootstrap/app.php';$app->make(\Illuminate\Contracts\Console\Kernel::class)->bootstrap();
}
namespace App\Services {
"""
probe=r"""
}
namespace {
use App\Models\Entity;use App\Services\Rat182PublicStatusReadOnlyProbe;use Illuminate\Support\Facades\DB;
DB::beginTransaction();DB::statement('SET TRANSACTION READ ONLY');DB::statement("SET LOCAL statement_timeout = '5s'");
try {
$p='01m0mw9hkbynssr7kbc0gk534j';$a='01m7x6ez57zpsgbv486y9t0y9h';$b='01m81mbmqvwr3d1nvfp9gkp3xf';$expected=['01m7x7kn332fqjxyj9b78zrmh0','01m81n8fsgxf1p52bw62e9rzyp'];sort($expected);
$service=app(Rat182PublicStatusReadOnlyProbe::class);
foreach([[$a,$b],[$b,$a]] as [$source,$target]){
 $entity=Entity::query()->with('type')->findOrFail($source);$groups=$service->discover($entity,[$p]);
 if(count($groups)!==1||$groups[0]['via']->id!==$p||count($groups[0]['items'])!==1||$groups[0]['items'][0]['entity']->id!==$target)throw new \RuntimeException('Existing canonical path not resolved');
 $path=$groups[0]['items'][0]['path'];$actual=array_column($path,'id');sort($actual);if($actual!==$expected)throw new \RuntimeException('Wrong persisted edges');
 if($groups[0]['via']->status!=='unverified'||array_unique(array_column($path,'verificationState'))!==['unverified'])throw new \RuntimeException('Verification status unexpectedly changed');
 echo 'CANONICAL_READONLY_REPAIR_PASS '.json_encode(['source'=>$source,'target'=>$target,'via'=>$p,'viaStatus'=>$groups[0]['via']->status,'path'=>$path],JSON_THROW_ON_ERROR)."\n";
}
echo "CANONICAL_QUERY_REPAIR_PROVED_WITH_NO_DATA_WRITES\n";
}finally{DB::rollBack();}
}
"""
Path(sys.argv[2]).write_text(bootstrap+s+probe)
CANARY
k -n rum-dev-isolated exec -i deploy/rum-api -c php-fpm -- php /dev/stdin <"$t/canary.php"
'''
marker='git diff --check; git add ';assert s.count(marker)==1;s=s.replace(marker,canary+'\n'+marker)
Path(sys.argv[2]).write_text(s)
MAKE
bash -n "$t/repair.sh";bash "$t/repair.sh"
