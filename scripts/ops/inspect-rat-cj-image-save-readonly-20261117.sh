#!/usr/bin/env bash
set -Eeuo pipefail
# CJ Investigates image-save diagnostic: strictly read-only, isolated canonical DEV.
# Filter away raw URL, user identity, session, IP and request bodies before relay output.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
ns='rum-dev-isolated'
expected='ghcr.io/daisycloversoftware/rum-api@sha256:5fc614b352a596e41d35a33f55b9f4b01d4c6c2bfbe2d7dda700d92267761a13'
actual="$(kctl -n "$ns" get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')"
[[ "$actual" == "$expected" ]] || { echo 'CJ_DIAG_API_DIGEST_GUARD_FAILED'; exit 78; }
[[ "$(kctl -n "$ns" get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == 'dev-rum-ra.daisycloversoftware.uk' ]] || exit 78
[[ "$(kctl -n rum-dev get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == 'rateurmate.online' ]] || exit 78
echo 'CJ_DIAG_SCOPE=canonical-isolated-dev-readonly'
# Canonical entity rights and image revision summary only: IDs, no accounts/URLs.
php_code='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$id="01m07d2f9fj05f4ypdbwp087qj";
$e=$db::table("entities")->where("id",$id)->first();
if(!$e){echo "CJ_ENTITY_NOT_FOUND\n";exit(66);}
$type=$db::table("entity_types")->where("id",$e->entity_type_id)->value("key");
$claims=$db::table("entity_claims")->where("entity_id",$id)->where("status","approved")->whereNull("revoked_at")->count();
$managers=$db::table("entity_managers")->where("entity_id",$id)->where("status","active")->whereNull("revoked_at")->count();
$revisions=$db::table("entity_profile_revisions")->where("entity_id",$id)->whereIn("field_key",["profile_image","cover_image"])->orderByDesc("occurred_at")->limit(10)->get();
print "CJ_CANONICAL_ENTITY=".json_encode(["exists"=>true,"id"=>$id,"type"=>$type,"status"=>$e->status,"visibility"=>$e->visibility,"publication"=>$e->publication_state,"editPolicy"=>$e->edit_policy,"editLocked"=>$e->edit_locked_at!==null,"claimCount"=>$claims,"managerCount"=>$managers,"imageRevisions"=>$revisions->map(static fn($r)=>["field"=>$r->field_key,"outcome"=>$r->outcome,"moderation"=>$r->moderation_status,"hasMedia"=>isset(json_decode($r->metadata??"{}",true)["mediaAssetId"])]),],JSON_UNESCAPED_SLASHES)."\n";
'
kctl -n "$ns" exec deployment/rum-api -c php-fpm -- php -r "$php_code"
# Selected nginx access logs from exact image-link endpoint, no IP/headers/body/URL.
kctl -n "$ns" logs deployment/rum-api -c nginx --since=8h --tail=30000 2>/dev/null | python3 -c '
import sys,re,collections
lines=sys.stdin
matches=[]
pat=re.compile(r"\"(POST) (/api/v1/rate-anything/things/([0-9a-z]+)/images/(link|upload))(?:\?[^ ]*)? HTTP/[0-9.]+\" ([0-9]{3})")
for line in lines:
 m=pat.search(line)
 if not m: continue
 status=int(m.group(5))
 if m.group(3)!="01m07d2f9fj05f4ypdbwp087qj": continue
 ts=re.search(r"\[([^\]]+)\]",line)
 matches.append(dict(at=(ts.group(1) if ts else None),route=m.group(2),status=status))
print("CJ_IMAGE_HTTP_RECORDS="+str(len(matches)))
print("CJ_IMAGE_HTTP_HISTOGRAM="+str(dict(collections.Counter(x["status"] for x in matches))))
for entry in matches[-30:]:print("CJ_IMAGE_HTTP="+str(entry))
'
# API PHP-FPM stderr may contain uncaught exception context. Sanitize then limit.
kctl -n "$ns" logs deployment/rum-api -c php-fpm --since=8h --tail=20000 2>/dev/null | python3 -c '
import sys,re
entries=[]
terms=re.compile(r"(?i)(thingimage|safethingimage|/images/link|yt3\.googleusercontent|guzzle|curl|media.?pipeline|\.ERROR|UnprocessableEntity|PDOException|QueryException|TypeError|ErrorException|Symfony.*Exception)")
for line in sys.stdin:
 if not terms.search(line):continue
 line=re.sub(r"https?://[^\s\"\\]+","[URL_REDACTED]",line)
 line=re.sub(r"\b[\w.+-]+@[\w.-]+\.[A-Za-z]{2,}\b","[EMAIL_REDACTED]",line)
 line=re.sub(r"(?i)(authorization|cookie|password|secret|token)(?:\s*[:=]\s*)[^\s,}]+",r"\1=[REDACTED]",line)
 line=re.sub(r"\b(?:\d{1,3}\.){3}\d{1,3}\b","[IP_REDACTED]",line)
 line=re.sub(r"(?i)(/var/www/html/)?(vendor/[^\s]+)",r"[FRAME_REDACTED]",line)
 if len(line)>440:line=line[:440]
 entries.append(line.strip())
print("CJ_PHP_RELEVANT_LOG_COUNT="+str(len(entries)))
for entry in entries[-45:]:print("CJ_PHP_LOG="+entry)
'
echo CJ_DIAGNOSTIC_READ_ONLY_COMPLETE=true
