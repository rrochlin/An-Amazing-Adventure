echo "testing default route output"
curl localhost:3000/api/test
echo ""

echo "testing dynamic url params"
curl localhost:3000/api/test/dynamic/334
echo ""

echo "testing query params"
curl "localhost:3000/api/test/query_params?v1=lksjdfl&v2=lsdkjfsdl"
echo ""
