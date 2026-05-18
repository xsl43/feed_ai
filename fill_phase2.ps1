# Fill comments, follows and messages - with rate limit handling
$b = 'http://localhost:8080'
$ct = @{'Content-Type' = 'application/json'}

function ApiCall($method, $url, $headers, $body) {
    $maxRetry = 3
    for ($i = 0; $i -lt $maxRetry; $i++) {
        try {
            return Invoke-RestMethod $url -Method $method -Headers $headers -Body $body
        } catch {
            $msg = $_.Exception.Message
            if ($msg -match "too many requests") {
                Write-Host "  Rate limited, waiting 3s..." -ForegroundColor DarkYellow
                Start-Sleep -Seconds 3
            } elseif ($msg -match "Duplicate") {
                return $null
            } else {
                throw
            }
        }
    }
    return $null
}

function Login($u) {
    return ApiCall "POST" "$b/account/login" $ct "{`"username`":`"$u`",`"password`":`"123456`"}"
}

# traveler_wang video 10 comments
Write-Host "Adding comments for traveler_wang videos..." -ForegroundColor Cyan
$l = Login "traveler_wang"
$h = @{'Authorization' = "Bearer $($l.token)"; 'Content-Type' = 'application/json'}
$vids = @(10, 11, 12)
$comments = @(
    @{v=10; c=@("So beautiful! How to plan a trip there?","Bucket list for sure! That green glow is surreal","Best aurora video ever! When is the best season?")},
    @{v=11; c=@("Erhai Lake is heaven on earth, going next week!","Whats the BGM? Love the vibes!","That sunset shot is absolutely insane")},
    @{v=12; c=@("Kyoto forever my favorite city in the world","Is it super crowded during leaf season?","Taking my mom there next year, bookmarking this")}
)
foreach ($c in $comments) {
    foreach ($t in $c.c) {
        ApiCall "POST" "$b/comment/publish" $h "{`"video_id`":$($c.v),`"content`":`"$t`"}" | Out-Null
        Start-Sleep -Milliseconds 500
    }
}
Write-Host "  traveler_wang comments done" -ForegroundColor Green

# foodie_li video 13-15 comments  
Write-Host "Adding comments for foodie_li videos..." -ForegroundColor Cyan
$l = Login "foodie_li"
$h = @{'Authorization' = "Bearer $($l.token)"; 'Content-Type' = 'application/json'}
$comments = @(
    @{v=13; c=@("My mouth is watering just watching this!","Where in Chengdu exactly? Need the address!","That chili oil looks absolutely perfect")},
    @{v=14; c=@("Braised pork is my ultimate weakness","Old restaurants just hit different, so authentic","That caramelized glaze is everything!")},
    @{v=15; c=@("Shrimp dumplings are the absolute best!","Saving this guide for my Guangzhou trip","Been to all 10 spots, great picks!")}
)
foreach ($c in $comments) {
    foreach ($t in $c.c) {
        ApiCall "POST" "$b/comment/publish" $h "{`"video_id`":$($c.v),`"content`":`"$t`"}" | Out-Null
        Start-Sleep -Milliseconds 500
    }
}
Write-Host "  foodie_li comments done" -ForegroundColor Green

# tech_ajie video 16-18 comments
Write-Host "Adding comments for tech_ajie videos..." -ForegroundColor Cyan
$l = Login "tech_ajie"
$h = @{'Authorization' = "Bearer $($l.token)"; 'Content-Type' = 'application/json'}
$comments = @(
    @{v=16; c=@("Finally a detailed comparison, about to buy a new phone!","Apple got destroyed this round honestly","Ordering a new phone based on this review")},
    @{v=17; c=@("Dream setup right there, what monitor is that?","That keyboard sounds so satisfying to type on","Developer goals material right here")},
    @{v=18; c=@("Tool number 3 is an absolute game changer","All free? That is incredible value","Already testing these out, productivity boosted!")}
)
foreach ($c in $comments) {
    foreach ($t in $c.c) {
        ApiCall "POST" "$b/comment/publish" $h "{`"video_id`":$($c.v),`"content`":`"$t`"}" | Out-Null
        Start-Sleep -Milliseconds 500
    }
}
Write-Host "  tech_ajie comments done" -ForegroundColor Green

# fitness_liu video 19-21 comments
Write-Host "Adding comments for fitness_liu videos..." -ForegroundColor Cyan
$l = Login "fitness_liu"
$h = @{'Authorization' = "Bearer $($l.token)"; 'Content-Type' = 'application/json'}
$comments = @(
    @{v=19; c=@("I have made all of these mistakes... oops","Been training wrong this entire time wow","Pure gold content, saved for later")},
    @{v=20; c=@("Followed along and feeling the burn everywhere!","Intense but so satisfying, day 1 done","Can beginners do this routine?")},
    @{v=21; c=@("Finally food that actually looks delicious while dieting","Screenshot saved for grocery shopping","3 pounds down this week following this!")}
)
foreach ($c in $comments) {
    foreach ($t in $c.c) {
        ApiCall "POST" "$b/comment/publish" $h "{`"video_id`":$($c.v),`"content`":`"$t`"}" | Out-Null
        Start-Sleep -Milliseconds 500
    }
}
Write-Host "  fitness_liu comments done" -ForegroundColor Green

# Follows - tester2026 follows all 4
Write-Host "Setting up follows..." -ForegroundColor Cyan
$l = Login "tester2026"
$h = @{'Authorization' = "Bearer $($l.token)"; 'Content-Type' = 'application/json'}
$userNames = @("traveler_wang", "foodie_li", "tech_ajie", "fitness_liu")
foreach ($u in $userNames) {
    $info = Invoke-RestMethod "$b/account/findByUsername" -Method POST -Headers $ct -Body "{`"username`":`"$u`"}"
    ApiCall "POST" "$b/social/follow" $h "{`"vlogger_id`":$($info.id)}" | Out-Null
    Write-Host "  tester2026 follows $u (ID=$($info.id))"
    Start-Sleep -Milliseconds 300
}

# Cross-follows between users
foreach ($u in $userNames) {
    $l = Login $u
    $h = @{'Authorization' = "Bearer $($l.token)"; 'Content-Type' = 'application/json'}
    foreach ($t in $userNames) {
        if ($t -ne $u) {
            $ti = Invoke-RestMethod "$b/account/findByUsername" -Method POST -Headers $ct -Body "{`"username`":`"$t`"}"
            ApiCall "POST" "$b/social/follow" $h "{`"vlogger_id`":$($ti.id)}" | Out-Null
            Start-Sleep -Milliseconds 300
        }
    }
    Write-Host "  $u follows all others" -ForegroundColor Green
    Start-Sleep -Milliseconds 500
}

# Messages
Write-Host "Sending messages..." -ForegroundColor Cyan
$tl = Login "tester2026"
$th = @{'Authorization' = "Bearer $($tl.token)"; 'Content-Type' = 'application/json'}
$msgs = @{
    "traveler_wang" = "Hey! Your Iceland video is absolutely stunning, looks like another planet!"
    "foodie_li" = "Love the hot pot episode! Making me so hungry right now"
    "tech_ajie" = "Great phone comparison! Really helped me decide what to buy"
    "fitness_liu" = "Coach! Been following your dumbbell routine for a week, feeling great!"
}
foreach ($u in $userNames) {
    $info = Invoke-RestMethod "$b/account/findByUsername" -Method POST -Headers $ct -Body "{`"username`":`"$u`"}"
    ApiCall "POST" "$b/message/send" $th "{`"to_id`":$($info.id),`"content`":`"$($msgs[$u])`"}" | Out-Null
    Write-Host "  tester2026 -> $u" -ForegroundColor Green
    Start-Sleep -Milliseconds 500
}

Write-Host "`n========== ALL DONE ==========" -ForegroundColor Green
