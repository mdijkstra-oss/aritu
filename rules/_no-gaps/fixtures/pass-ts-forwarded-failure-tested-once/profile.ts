export type User = {
  nickname: string | null;
  first: string;
  last: string;
};

export type UserStore = {
  load(id: string): Promise<User>;
};

export async function displayName(store: UserStore, id: string): Promise<string> {
  const user = await store.load(id);

  return user.nickname ?? `${user.first} ${user.last}`;
}
