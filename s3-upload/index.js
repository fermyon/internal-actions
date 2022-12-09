const core = require('@actions/core');
const exec = require('@actions/exec');

function splitBucketAndKey(dest) {
    const parts = dest.split('/');
    if (parts.length < 2) {
        throw new Error(`dest should be specified as '<bucket>/<key>'; got ${dest}`);
    }
    const bucket = parts.shift();
    const key = parts.join('/');
    return [bucket, key];
}

async function run() {
    try {
        const src = core.getInput('src');

        const env = {
            'AWS_REGION': core.getInput('region'),
            'AWS_ACCESS_KEY_ID': core.getInput('access-key-id'),
            'AWS_SECRET_ACCESS_KEY': core.getInput('secret-access-key'),
        };
        core.info(`Region: ${env['AWS_REGION']}`);

        // Parse copy dest(s) before upload
        const copies = core.getMultilineInput('copies').map(splitBucketAndKey);

        // Upload object
        const [bucket, key] = splitBucketAndKey(core.getInput('dest'));
        core.info(`Uploading ${src} to ${bucket}/${key}`);
        await exec.exec('aws', [
            's3api', 'put-object',
            '--bucket', bucket,
            '--key', key,
            '--body', src,
            '--acl', 'public-read',
        ], {env});
        core.setOutput("public-url", `https://${bucket}.s3.amazonaws.com/${key}`);

        // Make copies
        const copy_src = `${bucket}/${key}`;
        for (const [bucket, key] of copies) {
                core.info(`Copying ${copy_src} to ${bucket}/${key}`);
                await exec.exec('aws', [
                    's3api', 'copy-object',
                    '--copy-source', copy_src,
                    '--bucket', bucket,
                    '--key', key,
                    '--acl', 'public-read',
                ], {env});
        }
    } catch (err) {
        core.setFailed(err.message);
    }
}

run()