echo "testing default route output"
curl localhost:3000/api/test | jq '.'
echo ""

echo "testing dynamic url params"
curl localhost:3000/api/test/dynamic/334 | jq '.'
echo ""

echo "testing query params"
curl localhost:3000/api/test/query_params?"v1=lksjdfl&v2=lsdkjfsdl" | jq '.'
echo ""

echo "testing basic post"
curl -X POST localhost:3000/api/test \
    -H "Content-Type: application/json" \
    -d '{Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.KMUFsIDTnFmyG3nMiGM6H9FNFUROf3wh7SmqJp-QV30"}' |
    jq '.'
echo ""
