export function publishDigest(
  fetchPosts: () => { title: string; likes: number }[],
  render: (lines: string[]) => string,
  deliver: (body: string) => void,
): void {
  const posts = fetchPosts();
  const lines: string[] = [];
  for (const post of posts) {
    let stars = "";
    for (let step = 0; step < Math.min(5, Math.round(post.likes / 20)); step++) {
      stars += "*";
    }
    lines.push(`${post.title} ${stars}`);
  }
  deliver(render(lines));
}
