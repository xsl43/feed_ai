# Comprehensive API test script for feedsystem_ai_go
$b = 'http://localhost:8080'
$c = @{'Content-Type' = 'application/json'}
$pass = 0; $fail = 0
$log = "c:\Users\93983\Desktop\feed\api_test_log.txt"
"" | Out-File $log

function Test($name, $script) {
    try {
        $r = & $script
        if ($r) {
            $global:pass++
            "$name : PASS" | Tee-Object $log -Append
        } else {
            $global:fail++
            "$name : FAIL" | Tee-Object $log -Append
        }
    } catch {
        $global:fail++
        "$name : FAIL ($($_.Exception.Message.Substring(0,[Math]::Min(80,$_.Exception.Message.Length))))" | Tee-Object $log -Append
    }
}

Write-Host "========== ACCOUNT MODULE ==========" -ForegroundColor Cyan

# 1. Register new test user
$testUser = "apitest_" + (Get-Date -Format "HHmmss")
Test "1. Register" { Invoke-RestMethod "$b/account/register" -Method POST -Headers $c -Body "{`"username`":`"$testUser`",`"password`":`"test123`"}" | Out-Null; $true }

# 2. Login
$login = $null
Test "2. Login" { $script:login = Invoke-RestMethod "$b/account/login" -Method POST -Headers $c -Body "{`"username`":`"$testUser`",`"password`":`"test123`"}"; $login.token.Length -gt 10 }

# 3. Login - wrong password
Test "3. Login (wrong pwd)" { try { Invoke-RestMethod "$b/account/login" -Method POST -Headers $c -Body "{`"username`":`"$testUser`",`"password`":`"wrong`"}" | Out-Null; $false } catch { $true } }

# 4. Find by username
Test "4. findByUsername" { $r = Invoke-RestMethod "$b/account/findByUsername" -Method POST -Headers $c -Body "{`"username`":`"$testUser`"}"; $r.username -eq $testUser }

# 5. Find by ID
Test "5. findByID" { $r = Invoke-RestMethod "$b/account/findByID" -Method POST -Headers $c -Body "{`"id`":$($login.account_id)}"; $r.username -eq $testUser }

# 6. Get profile
Test "6. getProfile" { $r = Invoke-RestMethod "$b/account/getProfile" -Method POST -Headers $c -Body "{`"account_id`":$($login.account_id)}"; $r.account.username -eq $testUser }

# 7. Refresh token
$h = @{'Authorization' = "Bearer $($login.token)"; 'Content-Type' = 'application/json'}
Test "7. Refresh token" { $r = Invoke-RestMethod "$b/account/refresh" -Method POST -Headers $c -Body "{`"refresh_token`":`"$($login.refresh_token)`"}"; $r.token.Length -gt 10 }

# 8. Change password
Test "8. Change password" { Invoke-RestMethod "$b/account/changePassword" -Method POST -Headers $h -Body "{`"old_password`":`"test123`",`"new_password`":`"newpass456`"}" | Out-Null; $true }

# 9. Login with new password
Test "9. Login (new pwd)" { $r = Invoke-RestMethod "$b/account/login" -Method POST -Headers $c -Body "{`"username`":`"$testUser`",`"password`":`"newpass456`"}"; $script:login = $r; $r.token.Length -gt 10 }

# 10. Update profile
$h = @{'Authorization' = "Bearer $($login.token)"; 'Content-Type' = 'application/json'}
Test "10. Update profile" { Invoke-RestMethod "$b/account/updateProfile" -Method POST -Headers $h -Body "{`"bio`":`"API test user`",`"avatar_url`":`"/static/sample.jpg`"}" | Out-Null; $true }

# 11. Rename
Test "11. Rename" { Invoke-RestMethod "$b/account/rename" -Method POST -Headers $h -Body "{`"new_username`":`"$($testUser)_renamed`"}" | Out-Null; $true }

# 12. Logout
Test "12. Logout" { Invoke-RestMethod "$b/account/logout" -Method POST -Headers $h -Body "{}" | Out-Null; $true }

Write-Host "`n========== VIDEO MODULE ==========" -ForegroundColor Cyan

# Re-login after password change
$login = Invoke-RestMethod "$b/account/login" -Method POST -Headers $c -Body "{`"username`":`"$($testUser)_renamed`",`"password`":`"newpass456`"}"
$h = @{'Authorization' = "Bearer $($login.token)"; 'Content-Type' = 'application/json'}

# 13. Publish video
$vid = $null
Test "13. Publish video" { $script:vid = Invoke-RestMethod "$b/video/publish" -Method POST -Headers $h -Body "{`"title`":`"API Test Video`",`"description`":`"Testing video publish`",`"play_url`":`"/static/sample.mp4`",`"cover_url`":`"/static/sample.jpg`"}"; $vid.id -gt 0 }

# 14. Get video detail
Test "14. Get video detail" { $r = Invoke-RestMethod "$b/video/getDetail" -Method POST -Headers $c -Body "{`"id`":$($vid.id)}"; $r.title -eq "API Test Video" }

# 15. List by author ID
Test "15. List by author" { $r = Invoke-RestMethod "$b/video/listByAuthorID" -Method POST -Headers $c -Body "{`"author_id`":$($login.account_id)}"; $r.video_list.Count -gt 0 }

Write-Host "`n========== FEED MODULE ==========" -ForegroundColor Cyan

# 16. Latest feed
Test "16. Feed latest" { $r = Invoke-RestMethod "$b/feed/listLatest" -Method POST -Headers $c -Body '{"limit":10}'; $r.video_list.Count -gt 0 -and (Get-Member -InputObject $r -Name 'has_more') }

# 17. Popular feed
Test "17. Feed popular" { $r = Invoke-RestMethod "$b/feed/listByPopularity" -Method POST -Headers $c -Body '{"limit":10}'; $r.video_list.Count -gt 0 }

# 18. Following feed (auth required)
Test "18. Feed following" { $r = Invoke-RestMethod "$b/feed/listByFollowing" -Method POST -Headers $h -Body '{"limit":10}'; $r.video_list -ne $null }

# 19. LikesCount feed
Test "19. Feed likesCount" { $r = Invoke-RestMethod "$b/feed/listLikesCount" -Method POST -Headers $c -Body '{"limit":10}'; $r.video_list.Count -gt 0 }

Write-Host "`n========== SOCIAL MODULE ==========" -ForegroundColor Cyan

# 20. Like a video
$targetVid = 10
Test "20. Like video" { Invoke-RestMethod "$b/like/like" -Method POST -Headers $h -Body "{`"video_id`":$targetVid}" | Out-Null; $true }

# 21. Check is liked
Test "21. Is liked" { $r = Invoke-RestMethod "$b/like/isLiked" -Method POST -Headers $h -Body "{`"video_id`":$targetVid}"; $r.is_liked -eq $true }

# 22. Unlike
Test "22. Unlike" { Invoke-RestMethod "$b/like/unlike" -Method POST -Headers $h -Body "{`"video_id`":$targetVid}" | Out-Null; $true }

# 23. List my liked videos
Test "23. My liked videos" { $r = Invoke-RestMethod "$b/like/listMyLikedVideos" -Method POST -Headers $h -Body "{}"; $r.video_list -ne $null }

# 24. Follow testuser999 (ID=1)
Test "24. Follow user" { Invoke-RestMethod "$b/social/follow" -Method POST -Headers $h -Body '{"vlogger_id":1}' | Out-Null; $true }

# 25. Get follower count
Test "25. Follower count" { $r = Invoke-RestMethod "$b/social/getCounts" -Method POST -Headers $h -Body "{`"vlogger_id`":1}"; $r.follower_count -ge 0 }

# 26. Get all followers
Test "26. Get followers" { $r = Invoke-RestMethod "$b/social/getAllFollowers" -Method POST -Headers $h -Body "{`"vlogger_id`":1}"; $r.list -ne $null }

# 27. Get all vloggers (following)
Test "27. Get following" { $r = Invoke-RestMethod "$b/social/getAllVloggers" -Method POST -Headers $h -Body "{`"follower_id`":$($login.account_id)}"; $r.list -ne $null }

# 28. Unfollow
Test "28. Unfollow" { Invoke-RestMethod "$b/social/unfollow" -Method POST -Headers $h -Body '{"vlogger_id":1}' | Out-Null; $true }

Write-Host "`n========== COMMENT MODULE ==========" -ForegroundColor Cyan

# 29. Publish comment
$commentId = $null
Test "29. Publish comment" { $r = Invoke-RestMethod "$b/comment/publish" -Method POST -Headers $h -Body "{`"video_id`":$targetVid,`"content`":`"API test comment!`"}"; $script:commentId = $r.id; $r.id -gt 0 }

# 30. List comments
Test "30. List comments" { $r = Invoke-RestMethod "$b/comment/listAll" -Method POST -Headers $c -Body "{`"video_id`":$targetVid,`"page`":1}"; $r.list.Count -gt 0 }

# 31. Delete comment
Test "31. Delete comment" { Invoke-RestMethod "$b/comment/delete" -Method POST -Headers $h -Body "{`"comment_id`":$commentId}" | Out-Null; $true }

Write-Host "`n========== MESSAGE MODULE ==========" -ForegroundColor Cyan

# 32. Send message to traveler_wang (ID=13)
Test "32. Send message" { Invoke-RestMethod "$b/message/send" -Method POST -Headers $h -Body "{`"to_id`":13,`"content`":`"Hello from API test!`"}" | Out-Null; $true }

# 33. List messages
Test "33. List messages" { $r = Invoke-RestMethod "$b/message/list" -Method POST -Headers $h -Body "{`"peer_id`":13}"; $r.list -ne $null }

Write-Host "`n========== NOTIFICATION MODULE ==========" -ForegroundColor Cyan

# 34. Unread count
Test "34. Unread notifications" { $r = Invoke-RestMethod "$b/notification/unreadCount" -Method POST -Headers $h -Body "{}"; $r.count -ge 0 }

# 35. List notifications
Test "35. List notifications" { $r = Invoke-RestMethod "$b/notification/list" -Method POST -Headers $h -Body "{}"; $r.list -ne $null }

Write-Host "`n=========================================" -ForegroundColor Yellow
Write-Host "TOTAL: $pass PASS, $fail FAIL" -ForegroundColor $(if($fail -eq 0){'Green'}else{'Red'})
"`nTOTAL: $pass PASS, $fail FAIL" | Out-File $log -Append
