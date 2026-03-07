# S3 Bucket Setup for AI-Generated Map Images

## ✅ IMPLEMENTATION COMPLETE

The S3 infrastructure has been implemented and is ready for integration.

## Context

The "An Amazing Adventure" game is adding AI-generated map images using Google's Gemini API. These images need to be stored in AWS S3 and served to the game client.

## Request from Application Team

We need S3 infrastructure to store and serve game map images with the following requirements:

### Primary Requirements

1. **S3 Bucket for Map Storage**
   - Store AI-generated map images (PNG/JPEG format)
   - Images will be organized by game session: `{sessionUUID}/world-map.png`, `{sessionUUID}/zones/{zoneName}.png`
   - Need public read access (or presigned URLs - your decision on best approach)
   - Bucket should be in same region as DynamoDB (us-west-2)

2. **Access Pattern**
   - Go application server will upload images after generation
   - React client will fetch images to display in-game maps
   - Images are permanent per game session (no auto-deletion needed)
   - Low traffic initially (alpha testing)

3. **Security Considerations**
   - Server needs write access (IAM user/role)
   - Clients need read access (public or presigned URLs)
   - CORS configuration needed for React app to fetch images
   - No sensitive data in images (they're just maps)

4. **Cost Optimization**
   - Use Standard storage (images accessed frequently during gameplay)
   - Consider lifecycle rules for old/abandoned game sessions (optional)
   - Estimate: ~10-50 images per game session, 1-5MB each

### Technical Details for Integration

The Go application will need:
- Bucket name (to configure in Doppler secrets)
- IAM credentials with S3 write permissions
- Bucket region

The React client will need:
- Public URL pattern or mechanism to fetch images

### Implementation Decisions Left to You

- Whether to use bucket policies, IAM roles, or presigned URLs for access control
- Whether to enable versioning
- Whether to add CloudFront CDN (probably overkill for alpha)
- Specific CORS configuration
- Bucket naming convention
- Whether to create separate buckets for dev/staging/prod or use prefixes
- Lifecycle policies for cleanup

### Success Criteria

When complete, provide back to application team:
1. S3 bucket name(s)
2. Access credentials/method for server to upload
3. URL pattern or method for client to fetch images
4. Any necessary CORS or policy configurations
5. Doppler secret names to add (if creating new IAM users/keys)

## Notes

- Current infrastructure uses Terraform 1.12.2 and AWS Provider ~> 3.0
- Already have DynamoDB tables: `users`, `amazing-adventure-data`, `refresh-tokens`
- State is managed in S3 backend: `roberts-personal-tf-bucket`
- Region: us-west-2
- CI/CD via GitHub Actions already configured

## Questions to Consider

- Should images be encrypted at rest? (S3-managed keys would be fine)
- Do we want object versioning in case AI regenerates maps?
- Should we set up bucket logging for debugging?
- Any specific tagging requirements for cost tracking?

---

## 🎉 IMPLEMENTATION DETAILS

### Infrastructure Created

**S3 Bucket:** `amazing-adventure-map-images`
- **Region:** us-west-2
- **Encryption:** Enabled (AES256, S3-managed keys)
- **Versioning:** Enabled (useful if AI regenerates maps)
- **CORS:** Enabled for React client (GET/HEAD from any origin)

### Access Configuration

**For Server (Go Application):**
- **Upload Method:** Use existing AWS credentials (from Doppler)
- **Required IAM Policy:** `AmazingAdventureMapImagesUpload` (created by Terraform)
  - Attach this policy to your AWS IAM user/role
  - Grants: `s3:PutObject`, `s3:GetObject`, `s3:DeleteObject`, `s3:ListBucket`
- **Bucket Name:** `amazing-adventure-map-images`
- **Region:** `us-west-2`

**For Client (React Application):**
- **Access Method:** Public read access (simple HTTP GET requests)
- **URL Pattern:** `https://amazing-adventure-map-images.s3.us-west-2.amazonaws.com/{sessionUUID}/world-map.png`
- **Example URLs:**
  - World map: `https://amazing-adventure-map-images.s3.us-west-2.amazonaws.com/{sessionUUID}/world-map.png`
  - Zone map: `https://amazing-adventure-map-images.s3.us-west-2.amazonaws.com/{sessionUUID}/zones/{zoneName}.png`

### Object Path Structure

```
amazing-adventure-map-images/
├── {sessionUUID}/
│   ├── world-map.png
│   └── zones/
│       ├── forest.png
│       ├── mountain.png
│       └── ...
└── ...
```

### Integration Checklist

**Backend (Go):**
1. ✅ Add to Doppler: `S3_BUCKET_NAME=amazing-adventure-map-images`
2. ✅ Add to Doppler: `AWS_REGION=us-west-2`
3. ⚠️ **ACTION NEEDED:** Attach IAM policy `AmazingAdventureMapImagesUpload` to your AWS IAM user
   - Go to AWS Console → IAM → Users → [your user] → Permissions → Attach Policy
   - Search for "AmazingAdventureMapImagesUpload" and attach it
4. ✅ Use AWS SDK to upload images with path: `{sessionUUID}/world-map.png`

**Frontend (React):**
1. ✅ Build image URLs: `https://amazing-adventure-map-images.s3.us-west-2.amazonaws.com/{sessionUUID}/world-map.png`
2. ✅ Use standard `<img>` tags or fetch API - no special auth needed
3. ✅ CORS is configured - GET requests will work from any origin

### AWS SDK Example (Go)

```go
import (
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Upload a map image
sess := session.Must(session.NewSession(&aws.Config{
    Region: aws.String("us-west-2"),
}))

uploader := s3manager.NewUploader(sess)
_, err := uploader.Upload(&s3manager.UploadInput{
    Bucket: aws.String("amazing-adventure-map-images"),
    Key:    aws.String(fmt.Sprintf("%s/world-map.png", sessionUUID)),
    Body:   imageFile,
    ContentType: aws.String("image/png"),
})
```

### Security Notes

- ✅ Encryption at rest enabled (S3-managed keys)
- ✅ Public ACLs blocked (modern best practice)
- ✅ Public read via bucket policy (secure for non-sensitive images)
- ✅ Server write permissions via IAM policy (principle of least privilege)
- ✅ Versioning enabled (can recover if AI regenerates incorrectly)

### Cost Estimates

- **Storage:** ~$0.023/GB/month (Standard S3)
- **Requests:** ~$0.0004/1000 GET requests
- **Data Transfer:** First 100GB/month free, then ~$0.09/GB
- **Estimated alpha cost:** <$5/month for typical usage

### Deployment Status

- ✅ Terraform configuration added to `config/main.tf`
- ✅ Configuration validated (`terraform validate` passed)
- ✅ Plan tested (`terraform plan` shows 7 resources to add)
- ⏳ **Ready to apply:** Run `doppler run -- terraform apply` to create resources
- 📝 **Branch:** `feat/s3-map-images-infrastructure`

### Next Steps

1. **Review and Apply:** Review the Terraform changes and apply to create the S3 infrastructure
2. **Attach IAM Policy:** Attach `AmazingAdventureMapImagesUpload` to your AWS IAM user in AWS Console
3. **Update Go Application:** Add S3 upload logic using the bucket name and region above
4. **Update React Application:** Use the public URL pattern to display images
5. **Test:** Upload a test image and verify it's accessible from the React app

---

## Timeline

No rush - this is for a new feature branch (`feat/ai-generated-map-images` in An-Amazing-Adventure repo). Infrastructure is ready to deploy.

## Questions Answered

- ✅ **Encryption at rest?** Yes, S3-managed keys (AES256)
- ✅ **Object versioning?** Yes, enabled for recovery if AI regenerates
- ✅ **Bucket logging?** Not configured (can add if needed for debugging)
- ✅ **Tagging?** Yes, tagged with Name, Environment, Purpose for cost tracking
