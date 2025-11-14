# Infrastructure Dependencies

This file tracks infrastructure dependencies that are being set up in the terraform-infrastructure repository.

## Current Status: ✅ S3 Infrastructure Ready!

### S3 Bucket for AI-Generated Map Images

**Status**: ✅ **COMPLETE** - Ready for integration
**Branch**: `feat/ai-generated-map-images`
**Terraform Branch**: `feat/s3-map-images-infrastructure`

**Infrastructure Details**:
- **Bucket Name**: `amazing-adventure-map-images`
- **Region**: `us-west-2`
- **Encryption**: Enabled (AES256, S3-managed keys)
- **Versioning**: Enabled
- **CORS**: Configured for React client (GET/HEAD from any origin)
- **IAM Policy**: `AmazingAdventureMapImagesUpload`

**Configuration Received**:
1. ✅ S3 bucket name: `amazing-adventure-map-images`
2. ✅ Access method: Existing AWS credentials with new IAM policy
3. ✅ URL pattern: `https://amazing-adventure-map-images.s3.us-west-2.amazonaws.com/{sessionUUID}/world-map.png`
4. ✅ CORS: Configured for public GET requests
5. ✅ Doppler secrets needed: `S3_BUCKET_NAME`, `AWS_REGION`

**Actions Completed**:
1. ✅ Added AWS S3 SDK to Go dependencies (`server/go.mod`)
2. ✅ Created S3 client and upload functions (`server/s3.go`)
3. ✅ Added MapImages field to Game and SaveState structs
4. ✅ Updated NewGame, SaveGameState, and LoadGame functions

**Next Steps**:
1. ⚠️ **ACTION NEEDED**: Attach IAM policy `AmazingAdventureMapImagesUpload` to AWS IAM user in AWS Console
2. ⏳ Add to Doppler: `S3_BUCKET_NAME=amazing-adventure-map-images`
3. ⏳ Add to Doppler: `AWS_REGION=us-west-2` (if not already set)
4. ⏳ Integrate Gemini image generation API
5. ⏳ Update world generation to create and upload map images
6. ⏳ Update React client RoomMap component to display S3 images
7. ⏳ Test image generation and display flow

**Related Files Modified**:
- ✅ `server/s3.go` - New file with S3 client and upload functions
- ✅ `server/game.go` - Added MapImages field and persistence
- ⏳ `server/world_gen.go` - Will add image generation here
- ⏳ `server/mcp.go` - May need to add image generation tools
- ⏳ `client/src/components/RoomMap.tsx` - Will update to display images

---

## Future Infrastructure Needs

### CloudFront CDN (Optional)
- May add later for better image delivery performance
- Not needed for initial alpha testing

### Additional S3 Buckets (Maybe)
- Could separate dev/staging/prod environments
- Or use bucket prefixes instead

---

**Last Updated**: 2025-11-13
**Tracking Branch**: feat/ai-generated-map-images
