

# Releasing

This project includes Git tag driven automation to build and publish Github releases. When a release is created in github, this causes a tag to be added to the repository, which triggers a special tagged build in Travis. The build scripts run additional steps in particular on tagged builds. 

The outputs of the tagged build process include: 
* Docker images in the kabanero-io/kabanero-operator repository which are tagged matching the git tag
* Github release attachments including example operator deployments

## Preparing Sources

The sources may include references both to dependency versions as well as the current version of the software. These values need to be managed over the lifetime of the release, prior to the build/publish actions. 

Examples:
config/samples/full.yaml refers to the current recommended release number

Note: further details are coming in [issue 165](https://github.com/kabanero-io/kabanero-operator/issues/165)

## Publishing the Release

Once the sources are ready for final packaging and publishing. A Github Release can be created by following https://help.github.com/en/articles/creating-releases. 

The recommended publish procedure is as follows:

* Create one or more release candidates such as 0.1.1-RC.0 as pre-releases
* Verify the release candidate functions, fixes and behaviors, merging additional updates as required
* Create the final release only after verifying that all required fixes have been included. Check the pre-release box. 
* Update the release after final verification by unchecking the pre-release checkbox. 

### Use of Semantic Versioning

This project leverages semantic versioning for release naming and branching. Major and minor releases may be allocated a release branch within the source control repository and are also allocated a tag at release. Patch versions are only allocated a tag on delivery of the build since multiple revisions of the patch are not provided. When creating a release, it is important to select the correct release branch which corresponds with the release being delivered. 

### Creating a Release Candidate

Prior to creating the final release, a release candidate should be created. In the github release dialog, enter the release number followed by `-RC.N` where N is the candidate number. This follows the semantic versioning recommendations for release candidates. 

### Creating the final Release

To create the final release, open the new release editor and check the box "pre-release". Enter the version identifer such as `0.1.1` and use the same value for the tag. 

This will trigger the final release build. Pull the build result and perform any final verification, then return to the release editor dialog, uncheck the pre-release box, and save. The release is now complete. 