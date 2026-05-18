# Fill demo data for FeedAI platform - all English to avoid encoding issues
$b = 'http://localhost:8080'
$ct = @{'Content-Type' = 'application/json'}

function Login($u) {
    return Invoke-RestMethod "$b/account/login" -Method POST -Headers $ct -Body "{`"username`":`"$u`",`"password`":`"123456`"}"
}

function PublishVideo($h, $t, $d) {
    return Invoke-RestMethod "$b/video/publish" -Method POST -Headers $h -Body "{`"title`":`"$t`",`"description`":`"$d`",`"play_url`":`"/static/sample.mp4`",`"cover_url`":`"/static/sample.jpg`"}"
}

function AddComment($h, $vid, $c) {
    Invoke-RestMethod "$b/comment/publish" -Method POST -Headers $h -Body "{`"video_id`":$vid,`"content`":`"$c`"}" | Out-Null
}

Write-Host "=== Step 1: Create Users ===" -ForegroundColor Yellow
$userNames = @("traveler_wang", "foodie_li", "tech_ajie", "fitness_liu")
foreach ($u in $userNames) {
    try { Invoke-RestMethod "$b/account/register" -Method POST -Headers $ct -Body "{`"username`":`"$u`",`"password`":`"123456`"}" | Out-Null } catch {}
    Write-Host "  User ready: $u"
}

Write-Host "`n=== Step 2: Publish Videos for traveler_wang ===" -ForegroundColor Yellow
$l = Login "traveler_wang"
$h = @{'Authorization' = "Bearer $($l.token)"; 'Content-Type' = 'application/json'}
Invoke-RestMethod "$b/account/updateProfile" -Method POST -Headers $h -Body '{"bio":"Travel photographer | 50+ countries | Capturing the beauty of our world"}' | Out-Null
$tw_videos = @(
    @{t="Iceland Northern Lights - A Dream Come True"; d="Waited 4 hours at -20C for this magical aurora! Worth every freezing minute."; c=@("So beautiful! How to plan a trip?","Bucket list for sure! Already saving up.","The green glow is surreal","Best aurora video Ive ever seen!","When is the best season to go?")},
    @{t="Dali Erhai Lake Cycling - Pure Healing"; d="120km around Erhai Lake. Every frame is wallpaper-worthy. Yunnan is paradise."; c=@("Erhai Lake is heaven on earth","Going next week! Thanks for the route","Whats the BGM? Love it!","Yunnan never gets old","That sunset shot is insane")},
    @{t="Kyoto Red Leaves - Japan Autumn Magic"; d="Arashiyama + Kiyomizudera at night. Only 2 weeks a year for this stunning view."; c=@("Kyoto forever my favorite city","Is it crowded during leaf season?","Such a peaceful vibe","Taking my mom there next year","That last temple shot is perfection")}
)
foreach ($v in $tw_videos) {
    $pub = PublishVideo $h $v.t $v.d
    Write-Host "  Video: $($pub.id) - $($v.t)"
    foreach ($c in $v.c) { AddComment $h $pub.id $c }
    Start-Sleep -Milliseconds 150
}

Write-Host "`n=== Step 3: Publish Videos for foodie_li ===" -ForegroundColor Yellow
$l = Login "foodie_li"
$h = @{'Authorization' = "Bearer $($l.token)"; 'Content-Type' = 'application/json'}
Invoke-RestMethod "$b/account/updateProfile" -Method POST -Headers $h -Body '{"bio":"Food explorer | Street food hunter | Life is too short for bad meals"}' | Out-Null
$fl_videos = @(
    @{t="Chengdu Hidden Gem - Best Hot Pot Under $10"; d="Local-recommended hole-in-the-wall. 1 hour wait, 100% worth it! Sichuan spice heaven."; c=@("My mouth is watering just watching","Where in Chengdu exactly?","That chili oil looks perfect","Going to Chengdu next month, noted!","Can beginners handle the spice level?")},
    @{t="Shanghai Classic - 100 Year Old Braised Pork"; d="Recipe passed down 3 generations. The ultimate comfort food by the Bund."; c=@("Braised pork is my weakness","Old restaurants hit different","Been there, so good!","What is the price range?","That caramelized glaze is everything")},
    @{t="Guangzhou Morning Tea Guide - Top 10 Dim Sum Spots"; d="From Tao Tao Ju to Dian Du De. The ultimate Cantonese brunch experience."; c=@("Shrimp dumplings are the best!","Saving this for my Guangzhou trip","Ive been to all 10, great picks","Dim Sum culture is unmatched","Char siu bao forever")}
)
foreach ($v in $fl_videos) {
    $pub = PublishVideo $h $v.t $v.d
    Write-Host "  Video: $($pub.id) - $($v.t)"
    foreach ($c in $v.c) { AddComment $h $pub.id $c }
    Start-Sleep -Milliseconds 150
}

Write-Host "`n=== Step 4: Publish Videos for tech_ajie ===" -ForegroundColor Yellow
$l = Login "tech_ajie"
$h = @{'Authorization' = "Bearer $($l.token)"; 'Content-Type' = 'application/json'}
Invoke-RestMethod "$b/account/updateProfile" -Method POST -Headers $h -Body '{"bio":"Tech reviewer | Honest reviews only | No sponsors, just truth"}' | Out-Null
$ta_videos = @(
    @{t="2026 Flagship Phone Comparison - 5 Phones Tested"; d="Camera, battery, performance, display. Deep dive into the top 5 flagships of 2026."; c=@("Finally a detailed comparison!","About to buy a new phone, this helps","Apple got destroyed this round","Ordering based on this review","Best tech reviewer no cap")},
    @{t="Ultimate Programmer Desk Setup - $10K Workstation"; d="From monitors to keyboards, my complete productivity battlestation revealed."; c=@("Dream setup right there","What monitor is that?","That keyboard sounds amazing","So clean and minimal","Developer goals")},
    @{t="5 Free AI Tools That Will 10x Your Workflow"; d="All hands-on tested. Covers writing, coding, design, and automation. All free!"; c=@("Tool #3 is a game changer","All free?! Incredible","Productivity boosted 200%","Testing these out today","Any for video editing?")}
)
foreach ($v in $ta_videos) {
    $pub = PublishVideo $h $v.t $v.d
    Write-Host "  Video: $($pub.id) - $($v.t)"
    foreach ($c in $v.c) { AddComment $h $pub.id $c }
    Start-Sleep -Milliseconds 150
}

Write-Host "`n=== Step 5: Publish Videos for fitness_liu ===" -ForegroundColor Yellow
$l = Login "fitness_liu"
$h = @{'Authorization' = "Bearer $($l.token)"; 'Content-Type' = 'application/json'}
Invoke-RestMethod "$b/account/updateProfile" -Method POST -Headers $h -Body '{"bio":"NASM Certified Coach | Science-based fitness | No BS, just results"}' | Out-Null
$fl2_videos = @(
    @{t="Top 10 Beginner Fitness Mistakes - 90% People Make These"; d="From diet to training, avoid every common pitfall in your fitness journey."; c=@("Ive made all of these...","Been training wrong this whole time","Coach, how to lose belly fat?","Pure gold, bookmarking this","Starting my journey today!")},
    @{t="15 Minute Home Dumbbell HIIT - Full Body Burn"; d="No gym needed. Just one pair of dumbbells for a killer full-body workout."; c=@("Followed along, feeling the burn!","Intense but so satisfying","Can beginners do this?","Only have 5kg dumbbells, will it work?","Day 1 of this routine starts now")},
    @{t="Fat Loss Meal Plan - 7 Days of Healthy Eating"; d="High protein, low calorie, delicious. Never eat boring chicken breast again."; c=@("Finally food that actually looks good","Screenshot saved for grocery shopping","No more plain chicken breast!","3 pounds down this week","More recipes please!")}
)
foreach ($v in $fl2_videos) {
    $pub = PublishVideo $h $v.t $v.d
    Write-Host "  Video: $($pub.id) - $($v.t)"
    foreach ($c in $v.c) { AddComment $h $pub.id $c }
    Start-Sleep -Milliseconds 150
}

Write-Host "`n=== Step 6: Build Follow Network ===" -ForegroundColor Yellow
# Get all user IDs
$allIds = @()
foreach ($u in $userNames) {
    $info = Invoke-RestMethod "$b/account/findByUsername" -Method POST -Headers $ct -Body "{`"username`":`"$u`"}"
    $allIds += $info.id
    Write-Host "  $u -> ID: $($info.id)"
}

# Each user follows all other users
foreach ($u in $userNames) {
    $l = Login $u
    $h = @{'Authorization' = "Bearer $($l.token)"; 'Content-Type' = 'application/json'}
    foreach ($tid in $allIds) {
        if ($tid -ne $l.account_id) {
            Invoke-RestMethod "$b/social/follow" -Method POST -Headers $h -Body "{`"vlogger_id`":$tid}" | Out-Null
        }
    }
    Write-Host "  $u follows everyone"
    Start-Sleep -Milliseconds 150
}

# tester2026 follows all 4 users
Write-Host "  tester2026 follows everyone"
$l = Login "tester2026"
if (-not $l.token) { 
    try { Invoke-RestMethod "$b/account/register" -Method POST -Headers $ct -Body '{"username":"tester2026","password":"pass123"}' | Out-Null } catch {}
    $l = Invoke-RestMethod "$b/account/login" -Method POST -Headers $ct -Body '{"username":"tester2026","password":"pass123"}'
}
$h = @{'Authorization' = "Bearer $($l.token)"; 'Content-Type' = 'application/json'}
foreach ($tid in $allIds) {
    Invoke-RestMethod "$b/social/follow" -Method POST -Headers $h -Body "{`"vlogger_id`":$tid}" | Out-Null
}

Write-Host "`n=== Step 7: Send Messages ===" -ForegroundColor Yellow
$messages = @(
    "Hey! Love your content, especially the recent video!",
    "Your shots are incredible! What camera do you use?",
    "Great review! Helped me decide on my next phone, thanks!",
    "Coach, your HIIT routine is killer! Already seeing results after a week!"
)
$testerLogin = Invoke-RestMethod "$b/account/login" -Method POST -Headers $ct -Body '{"username":"tester2026","password":"pass123"}'
$testerId = $testerLogin.account_id
for ($i = 0; $i -lt $messages.Count; $i++) {
    $u = $userNames[$i]
    $l = Login $u
    $h = @{'Authorization' = "Bearer $($l.token)"; 'Content-Type' = 'application/json'}
    Invoke-RestMethod "$b/message/send" -Method POST -Headers $h -Body "{`"to_id`":$testerId,`"content`":`"$($messages[$i])`"}" | Out-Null
    Write-Host "  $u -> tester2026: $($messages[$i])"
    Start-Sleep -Milliseconds 100
}

Write-Host "`n========== DONE! Data populated successfully ==========" -ForegroundColor Green
