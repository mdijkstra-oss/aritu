export type Status = "draft" | "inReview" | "approved" | "published" | "archived";

export type Action = "submit" | "approve" | "reject" | "publish" | "archive" | "revise" | "restore";

export function nextStatus(current: Status, action: Action): Status {
  const target = transitions[`${current}:${action}`];
  if (target === undefined) {
    throw new Error(`cannot ${action} from ${current}`);
  }

  return target;
}

const transitions: Record<string, Status | undefined> = {
  "draft:submit": "inReview",
  "draft:archive": "archived",
  "inReview:approve": "approved",
  "inReview:reject": "draft",
  "inReview:archive": "archived",
  "approved:publish": "published",
  "approved:revise": "draft",
  "published:archive": "archived",
  "archived:restore": "draft",
};
