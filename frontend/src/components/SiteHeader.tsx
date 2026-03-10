import { Link } from "react-router-dom";
import { useAuth } from "../contexts/AuthContext";
import { useAuthModal } from "../contexts/AuthModalContext";

export interface SiteHeaderProps {
  showPostsLink?: boolean;
  editLink?: { to: string; label: string } | null;
}

export default function SiteHeader({ showPostsLink, editLink }: SiteHeaderProps) {
  const { accessToken } = useAuth();
  const { openLoginModal, openRegisterModal } = useAuthModal();

  return (
    <header className="border-b bg-white">
      <div className="max-w-3xl mx-auto px-4 py-4 flex items-center justify-between">
        <Link to="/" className="font-semibold text-gray-800">
          Blogedit
        </Link>
        <nav className="flex items-center gap-4">
          {showPostsLink && (
            <Link to="/" className="text-gray-600 hover:underline">
              Posts
            </Link>
          )}
          {accessToken ? (
            <>
              <Link to="/editor" className="text-blue-600 hover:underline">
                Write
              </Link>
              {editLink && (
                <Link to={editLink.to} className="text-blue-600 hover:underline">
                  {editLink.label}
                </Link>
              )}
              <span className="text-gray-500 text-sm">Logged in</span>
            </>
          ) : (
            <>
              <button type="button" onClick={openLoginModal} className="text-gray-600 hover:underline">
                Login
              </button>
              <button type="button" onClick={openRegisterModal} className="text-gray-600 hover:underline">
                Register
              </button>
            </>
          )}
        </nav>
      </div>
    </header>
  );
}
