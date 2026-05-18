$b = 'http://localhost:8080'
$c = @{'Content-Type' = 'application/json'}
$log = "c:\Users\93983\Desktop\feed\fill_log.txt"
"=== Follow & Message Fill Started ===" | Out-File $log

$users = @("traveler_wang", "foodie_li", "tech_ajie", "fitness_liu")
$ids = @{}
foreach ($u in $users) {
    $r = Invoke-RestMethod "$b/account/findByUsername" -Method POST -Headers $c -Body "{`"username`":`"$u`"}"
    $ids[$u] = $r.id
    "$u = ID $($r.id)" | Out-File $log -Append
}
"IDs collected: $($ids.Count)" | Out-File $log -Append

# Login & get token helper
function GetToken($u) {
    $r = Invoke-RestMethod "$b/account/login" -Method POST -Headers $c -Body "{`"username`":`"$u`",`"password`":`"123456`"}"
    return $r.token
}

# Step 1: Cross-follows - each user follows all others
foreach ($u in $users) {
    Start-Sleep -Milliseconds 300
    $token = GetToken $u
    $h = @{'Authorization' = "Bearer $token"; 'Content-Type' = 'application/json'}
    foreach ($t in $users) {
        if ($t -ne $u) {
            $tid = $ids[$t]
            try {
                $r = Invoke-RestMethod "$b/social/follow" -Method POST -Headers $h -Body "{`"vlogger_id`":$tid}"
                "$u -> follow $t (ID=$tid) : OK" | Out-File $log -Append
            } catch {
                "$u -> follow $t (ID=$tid) : $_" | Out-File $log -Append
            }
            Start-Sleep -Milliseconds 300
        }
    }
    "$u done following" | Out-File $log -Append
}

# Step 2: Also follow testuser999 (ID=1) 
$token = GetToken "traveler_wang"
$h = @{'Authorization' = "Bearer $token"; 'Content-Type' = 'application/json'}
try { Invoke-RestMethod "$b/social/follow" -Method POST -Headers $h -Body '{"vlogger_id":1}' | Out-Null; "traveler_wang follows testuser999" | Out-File $log -Append } catch {}
$token = GetToken "foodie_li"; $h = @{'Authorization' = "Bearer $token"; 'Content-Type' = 'application/json'}
try { Invoke-RestMethod "$b/social/follow" -Method POST -Headers $h -Body '{"vlogger_id":1}' | Out-Null; "foodie_li follows testuser999" | Out-File $log -Append } catch {}
$token = GetToken "tech_ajie"; $h = @{'Authorization' = "Bearer $token"; 'Content-Type' = 'application/json'}
try { Invoke-RestMethod "$b/social/follow" -Method POST -Headers $h -Body '{"vlogger_id":1}' | Out-Null; "tech_ajie follows testuser999" | Out-File $log -Append } catch {}
$token = GetToken "fitness_liu"; $h = @{'Authorization' = "Bearer $token"; 'Content-Type' = 'application/json'}
try { Invoke-RestMethod "$b/social/follow" -Method POST -Headers $h -Body '{"vlogger_id":1}' | Out-Null; "fitness_liu follows testuser999" | Out-File $log -Append } catch {}

# Step 3: Messages between users
$msgs = @{
    "traveler_wang" = @{to="foodie_li"; txt="Your hot pot video is making me so hungry! Any recommendations for best spots in Chengdu?"}
    "foodie_li" = @{to="traveler_wang"; txt="Love your Iceland shots! I went there last year and your video brought back great memories"}
    "tech_ajie" = @{to="fitness_liu"; txt="Coach, what fitness tracker do you recommend? Looking to upgrade my gear"}
    "fitness_liu" = @{to="tech_ajie"; txt="Great review on the flagship phones! Can you review fitness wearables next?"}
}
foreach ($msg in $msgs.GetEnumerator()) {
    Start-Sleep -Milliseconds 400
    $token = GetToken $msg.Key
    $h = @{'Authorization' = "Bearer $token"; 'Content-Type' = 'application/json'}
    $toId = $ids[$msg.Value.to]
    try {
        $r = Invoke-RestMethod "$b/message/send" -Method POST -Headers $h -Body "{`"to_id`":$toId,`"content`":`"$($msg.Value.txt)`"}"
        "$($msg.Key) -> $($msg.Value.to) : OK" | Out-File $log -Append
    } catch {
        "$($msg.Key) -> $($msg.Value.to) : $_" | Out-File $log -Append
    }
}

"=== Fill Complete ===" | Out-File $log -Append
