import { Link } from "react-router";

interface NavbarButtonProps {
  title: string;
  link: string;
  options: NavbarOption[];
  img: string;
}

type NavbarOption = {
  title: string;
  link: string;
  img: string;
};

function NavbarButton({ title, link, options, img }: NavbarButtonProps) {
  return (
    <div className="relative group w-full flex justify-center rounded-sm">
      <Link
        to={link}
        className="flex gap-2 pl-1 items-center w-9/10 h-9 text-md text-white/90 font-bold hover:cursor-pointer hover:bg-bg rounded-sm"
      >
        <img src={img} className="size-5" />
        {title}
      </Link>
      <div className="absolute left-full -top-1.5 translate-y-0 px-1.5 py-1.5 hidden group-hover:flex flex-col w-70 bg-nav-sub-bg rounded-r-sm">
        {options.map((option) => (
          <Link
            to={option.link}
            className="flex items-center gap-2 pl-1 h-9 text-sm text-white/90 font-bold hover:cursor-pointer hover:bg-nav-sub-hover rounded-sm relative z-50"
          >
            <img src={option.img} className="size-5" />
            {option.title}
          </Link>
        ))}
      </div>
    </div>
  );
}

export default NavbarButton;
