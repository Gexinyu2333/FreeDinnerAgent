export type PublicUser = {
  id: string;
  username: string;
  display_name: string | null;
};

export type AuthResult = {
  access_token: string;
  refresh_token: string;
  user: PublicUser;
};

export type LoginInput = {
  username: string;
  password: string;
};

export type RegisterInput = {
  username: string;
  password: string;
  display_name?: string;
};
