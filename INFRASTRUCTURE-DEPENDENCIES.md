# Infrastructure Dependencies

This file tracks infrastructure dependencies that are being set up in the terraform-infrastructure repository.

## Current Status: ⏳ Waiting for S3 Setup

### S3 Bucket for AI-Generated Map Images

**Status**: Requested - awaiting implementation
**Branch**: `feat/ai-generated-map-images`
**Terraform TODO**: See `/home/rob/repos/terraform-infrastructure/CLAUDE-TODO.md`

**What We Need**:
- S3 bucket for storing map images
- IAM credentials for server upload access
- Access method for client to fetch images (public URLs or presigned)
- CORS configuration for React app

**What We're Waiting For**:
1. S3 bucket name(s)
2. IAM credentials or access method for server
3. URL pattern for client image fetching
4. Any CORS/policy configurations needed
5. Doppler secret names to configure

**Next Steps After Infrastructure is Ready**:
1. Add S3 bucket name to Doppler secrets
2. Add IAM credentials to Doppler (if using IAM user)
3. Install AWS S3 SDK in Go server (`server/go.mod`)
4. Implement image upload functions in Go
5. Integrate Gemini image generation API
6. Update React client to display S3-hosted images

**Related Files in This Repo**:
- `server/world_gen.go` - Will add image generation logic here
- `client/src/components/RoomMap.tsx` - Will update to display images
- Server will need new files for S3 integration

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
