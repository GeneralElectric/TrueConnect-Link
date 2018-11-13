$clientId = ""
$clientSecret = ""

do {
    $tokResp = Invoke-WebRequest -Uri https://a8a2ffc4-b04e-4ec1-bfed-7a51dd408725.predix-uaa.run.aws-usw02-pr.ice.predix.io/oauth/token -Body ("grant_type=client_credentials&client_id="+$clientId+"&client_secret="+$clientSecret+"&response_type=token") -ContentType application/x-www-form-urlencoded -Method POST | ConvertFrom-Json

    $headers = @{}
    $headers.Add("authorization","Bearer "+$tokResp.access_token)

    $uploads = (Invoke-WebRequest -Method GET -Uri "https://trueconnect.run.aws-usw02-pr.ice.predix.io/api/v1/files" -Headers $headers).Content | ConvertFrom-Json

    $index = 0
    

    $uploads | foreach {
        $current = $_
        "$index : " + $current.metadata.original_file_name.value + "  :  " + $current.data_store_ref +"  :  "+$current.metadata.file_arrival_date.value
        $index++

    }

    $index = Read-host "Which file do you want to download"    

    $headers = @{}
    $headers.Add("authorization","Bearer "+$tokResp.access_token)

    $fileName = [System.IO.Path]::GetFileName($uploads[$index].metadata.original_file_name.value)

    $fileName = ".\Bombardier\"+$uploads[$index].data_store_ref+"\"+$fileName

    $folder = [System.IO.Path]::GetDirectoryName($fileName)

    if (Test-Path $folder) {
        "you already have this one"
    } else {
        mkdir $folder
        Invoke-WebRequest -Uri ("https://trueconnect.run.aws-usw02-pr.ice.predix.io/api/v1/files/"+$uploads[$index].data_store_ref) -Method GET -OutFile $fileName -Headers $headers
        $fileName
    }

    $again = Read-Host "Again? [y/n]"
}
While ($again -eq "y")